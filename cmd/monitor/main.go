package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/numbergroup/eth-monitor/pkg/alert"
	"github.com/numbergroup/eth-monitor/pkg/config"
	"github.com/numbergroup/eth-monitor/pkg/monitor"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	confFile := flag.String("conf", "./config.yaml", "path to the configuration file")

	flag.Parse()

	conf, err := config.LoadConfig(*confFile)
	if err != nil {
		print(err.Error())
		os.Exit(1)
	}

	waitGroup := &sync.WaitGroup{}
	conf.Log.WithField("endpoints", len(conf.Endpoints)).Info("starting monitors")
	for _, endpoint := range conf.Endpoints {
		alertChannels := []alert.Alert{}
		if endpoint.Pagerduty.Enabled {
			alertChannels = append(alertChannels, alert.NewPagerduty(conf, endpoint))
		}
		if endpoint.Slack.Enabled {
			alertChannels = append(alertChannels, alert.NewSlack(conf, endpoint))
		}

		rpcClient, err := ethclient.DialContext(ctx, endpoint.URL)
		if err != nil {
			conf.Log.WithError(err).WithField("endpoint", endpoint.Name).Panic("failed to connect to RPC client")
		}
		mon, err := monitor.NewBlockNumberMonitor(conf, alertChannels, rpcClient, endpoint)
		if err != nil {
			conf.Log.WithError(err).WithField("endpoint", endpoint.Name).Panic("failed to create monitor")
		}
		waitGroup.Add(1)
		go func(m monitor.Monitor) {
			conf.Log.WithField("name", m.Name()).Info("starting monitoring")

			defer waitGroup.Done()
			m.Run(ctx)
		}(mon)

		// Start PeerCount monitor if configured
		if endpoint.MinPeers > 0 {
			peerMon, err := monitor.NewPeerCountMonitor(conf, alertChannels, rpcClient, endpoint)
			if err != nil {
				conf.Log.WithError(err).WithField("endpoint", endpoint.Name).Panic("failed to create peer monitor")
			}
			waitGroup.Add(1)
			go func(m monitor.Monitor) {
				conf.Log.WithField("name", m.Name()).Info("starting monitoring")

				defer waitGroup.Done()
				m.Run(ctx)
			}(peerMon)
		}

	}

	waitGroup.Wait()
	conf.Log.Info("all monitors stopped, exiting")
}
