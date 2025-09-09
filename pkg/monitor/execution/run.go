package execution

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/numbergroup/eth-monitor/pkg/alert"
	"github.com/numbergroup/eth-monitor/pkg/config"
	"github.com/numbergroup/eth-monitor/pkg/monitor"
)

func RunMonitors(ctx context.Context, waitGroup *sync.WaitGroup, conf *config.Config, endpoint config.Endpoint, alertChannels []alert.Alert) error {
	rpcClient, err := ethclient.DialContext(ctx, endpoint.URL)
	if err != nil {
		conf.Log.WithError(err).WithField("endpoint", endpoint.Name).Error("failed to connect to RPC client")
		return err
	}
	mon, err := NewBlockNumberMonitor(conf, alertChannels, rpcClient, endpoint)
	if err != nil {
		conf.Log.WithError(err).WithField("endpoint", endpoint.Name).Error("failed to create block number monitor")
		return err
	}
	waitGroup.Add(1)
	go func(m monitor.Monitor) {
		conf.Log.WithField("name", m.Name()).Info("block number monitoring started")

		defer waitGroup.Done()
		m.Run(ctx)
	}(mon)

	// Start PeerCount monitor if configured
	if endpoint.MinPeers > 0 {
		peerMon, err := NewPeerCountMonitor(conf, alertChannels, rpcClient, endpoint)
		if err != nil {
			conf.Log.WithError(err).WithField("endpoint", endpoint.Name).Error("failed to create peer monitor")
			return err
		}
		waitGroup.Add(1)
		go func(m monitor.Monitor) {
			conf.Log.WithField("name", m.Name()).Info("peer count monitoring started")

			defer waitGroup.Done()
			m.Run(ctx)
		}(peerMon)
	}
	return nil
}
