package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/numbergroup/eth-monitor/pkg/config"
)

func main() {
	quitCh := make(chan os.Signal, 1)
	signal.Notify(quitCh, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		<-quitCh
	}()

	confFile := *flag.String("conf", "./config.yaml", "path to the configuration file")
	conf, err := config.LoadConfig(confFile)
	if err != nil {
		print(err.Error())
		os.Exit(1)
	}

	waitGroup := &sync.WaitGroup{}

	for _, endpoint := range conf.Endpoints {
		mon, err := NewMonitor(ctx, conf, endpoint)
		if err != nil {
			conf.Log.WithError(err).WithField("endpoint", endpoint.Name).Panic("failed to create monitor")
		}

		go func(m *monitor) {
			m.conf.Log.WithField("name", m.endpoint.Name).Info("starting monitoring")
			waitGroup.Add(1)
			defer waitGroup.Done()
			m.Run(ctx)
		}(mon)

	}

	waitGroup.Wait()
	conf.Log.Info("all monitors stopped, exiting")
	cancel() // Ensure context is cancelled to stop all goroutines gracefully
}
