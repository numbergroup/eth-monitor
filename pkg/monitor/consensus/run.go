package consensus

import (
	"context"
	"sync"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/cockroachdb/errors"

	"github.com/numbergroup/eth-monitor/pkg/alert"
	"github.com/numbergroup/eth-monitor/pkg/config"
)

func RunMonitors(ctx context.Context, waitGroup *sync.WaitGroup, conf *config.Config, endpoint config.Endpoint, alertChannels []alert.Alert) error {
	client, err := http.New(ctx, http.WithAddress(endpoint.URL))
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP client")
	}
	httpClient := client.(*http.Service)

	if endpoint.NewBlockMaxDuration > 0 {
		waitGroup.Add(1)
		go func() {
			mon := NewBlockMonitor(conf, httpClient, endpoint, alertChannels)
			conf.Log.WithField("name", mon.Name()).Info("block monitoring started")

			defer waitGroup.Done()
			mon.Run(ctx)
		}()
	}

	if endpoint.MinPeers > 0 {
		waitGroup.Add(1)
		go func() {
			mon, err := NewPeerCountMonitor(conf, alertChannels, endpoint)
			if err != nil {
				conf.Log.WithError(err).WithField("endpoint", endpoint.Name).Error("failed to create peer count monitor")
				return
			}
			conf.Log.WithField("name", mon.Name()).Info("peer count monitoring started")

			defer waitGroup.Done()
			mon.Run(ctx)
		}()
	}

	return nil
}
