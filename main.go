package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/alecthomas/kingpin"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/oklog/run"
	"github.com/prometheus/common/promlog"
)

var (
	// groupLoader    = rules.FileLoader{}
	// filename       = "rules.yaml"
	// interval       = 10 * time.Second
	standaloneMode = kingpin.Flag("standalone", "Enable standalone mode, used for out of a K8s cluster.").Default("false").Bool()
)

func init() {
	fmt.Println("main standaloneMode: " + strconv.FormatBool(*standaloneMode))
}

func main() {
	logger := promlog.New(&promlog.Config{})
	ctxWeb, cancelWeb := context.WithCancel(context.Background())

	webHandler := NewHandler(log.With(logger, "component", "web"))
	listener, err := webHandler.Listener()
	if err != nil {
		level.Error(logger).Log("msg", "Unable to start web listener", "err", err)
		os.Exit(1)
	}

	var g run.Group
	{
		// Web handler.
		g.Add(
			func() error {
				if err := webHandler.Run(ctxWeb, listener, ""); err != nil {
					return fmt.Errorf("error starting web server: %w", err)
				}
				return nil
			},
			func(err error) {
				cancelWeb()
			},
		)
	}
	if err := g.Run(); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "See you next time!")
}
