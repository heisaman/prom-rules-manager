package main

import (
	"context"
	"encoding/json"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/mwitkow/go-conntrack"
	"github.com/prometheus/common/route"
	toolkit_web "github.com/prometheus/exporter-toolkit/web"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/net/netutil"
)

const (
	ListenAddress  = "0.0.0.0:9090"
	MaxConnections = 512
)

// withStackTrace logs the stack trace in case the request panics. The function
// will re-raise the error which will then be handled by the net/http package.
// It is needed because the go-kit log package doesn't manage properly the
// panics from net/http (see https://github.com/go-kit/kit/issues/233).
func withStackTracer(h http.Handler, l log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				level.Error(l).Log("msg", "panic while serving request", "client", r.RemoteAddr, "url", r.URL, "err", err, "stack", buf)
				panic(err)
			}
		}()
		h.ServeHTTP(w, r)
	})
}

// Handler serves various HTTP endpoints of the Prometheus server
type Handler struct {
	logger log.Logger

	context context.Context
	router  *route.Router
	quitCh  chan struct{}
	birth   time.Time
	cwd     string

	mtx sync.RWMutex
}

// New initializes a new web Handler.
func NewHandler(logger log.Logger) *Handler {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	router := route.New()

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "<error retrieving current working directory>"
	}

	h := &Handler{
		logger: logger,
		router: router,
		cwd:    cwd,
	}

	router.Post("/api/rules/add", func(w http.ResponseWriter, r *http.Request) {
		level.Info(h.logger).Log("msg", "Add rules...")
		decoder := json.NewDecoder(r.Body)
		var ruleGroup SimpleRuleGroup
		err := decoder.Decode(&ruleGroup)
		if err != nil {
			level.Error(h.logger).Log("msg", fmt.Sprintf("Error decoding request body: %s", err))
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Group rules cannot be decoded.\n")
			return
		}

		rulesManager := NewRulesManager()
		rulesManager.AddRules(ruleGroup)

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Rules are added successfully.\n")
	})
	router.Post("/api/rules/delete", func(w http.ResponseWriter, r *http.Request) {
		level.Info(h.logger).Log("msg", "Delete rules...")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Rules are deleted successfully.\n")
	})

	return h
}

// Listener creates the TCP listener for web requests.
func (h *Handler) Listener() (net.Listener, error) {
	level.Info(h.logger).Log("msg", "Start listening for connections", "address", ListenAddress)

	listener, err := net.Listen("tcp", ListenAddress)
	if err != nil {
		return listener, err
	}
	listener = netutil.LimitListener(listener, MaxConnections)

	// Monitor incoming connections with conntrack.
	listener = conntrack.NewListener(listener,
		conntrack.TrackWithName("http"),
		conntrack.TrackWithTracing())

	return listener, nil
}

// Run serves the HTTP endpoints.
func (h *Handler) Run(ctx context.Context, listener net.Listener, webConfig string) error {
	if listener == nil {
		var err error
		listener, err = h.Listener()
		if err != nil {
			return err
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/", h.router)

	errlog := stdlog.New(log.NewStdlibAdapter(level.Error(h.logger)), "", 0)

	spanNameFormatter := otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
		return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
	})

	httpSrv := &http.Server{
		Handler:     withStackTracer(otelhttp.NewHandler(mux, "", spanNameFormatter), h.logger),
		ErrorLog:    errlog,
		ReadTimeout: time.Duration(0),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- toolkit_web.Serve(listener, httpSrv, &toolkit_web.FlagConfig{WebConfigFile: &webConfig}, h.logger)
	}()

	select {
	case e := <-errCh:
		return e
	case <-ctx.Done():
		httpSrv.Shutdown(ctx)
		return nil
	}
}
