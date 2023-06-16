package main

import (
	"context"
	"net"
	"os"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/mwitkow/go-conntrack"
	"github.com/prometheus/common/route"
	"golang.org/x/net/netutil"
)

const (
	ListenAddress  = "0.0.0.0:9090"
	MaxConnections = 512
)

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
