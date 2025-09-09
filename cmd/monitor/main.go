package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/numbergroup/eth-monitor/pkg/alert"
	"github.com/numbergroup/eth-monitor/pkg/config"
	"github.com/numbergroup/eth-monitor/pkg/monitor/consensus"
	"github.com/numbergroup/eth-monitor/pkg/monitor/execution"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	confFile := flag.String("conf", "./config.yaml", "path to the configuration file")

	var (
		conf *config.Config
		err  error
	)
	flag.Parse()
	if confFile == nil || *confFile == "" {
		conf, err = config.LoadConfig([]byte(os.Getenv("ETH_MONITOR_CONFIG_DATA")))
		if err != nil {
			panic(err)
		}
	} else {
		conf, err = config.LoadConfigFromFile(*confFile)
		if err != nil {
			panic(err)
		}
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

		switch endpoint.Type {
		case config.TypeExecution:
			err := execution.RunMonitors(ctx, waitGroup, conf, endpoint, alertChannels)
			if err != nil {
				conf.Log.WithError(err).WithField("endpoint", endpoint.Name).Panic("failed to run monitors")
			}
		case config.TypeConsensus:
			err := consensus.RunMonitors(ctx, waitGroup, conf, endpoint, alertChannels)
			if err != nil {
				conf.Log.WithError(err).WithField("endpoint", endpoint.Name).Panic("failed to run monitors")
			}
		}

	}

	waitGroup.Wait()
	conf.Log.Info("all monitors stopped, exiting")
}
