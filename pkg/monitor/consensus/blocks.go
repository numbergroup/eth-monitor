package consensus

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/api"
	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"

	"github.com/numbergroup/eth-monitor/pkg/alert"
	"github.com/numbergroup/eth-monitor/pkg/config"
	"github.com/numbergroup/eth-monitor/pkg/monitor"
)

type BlockMonitorETH2RPC interface {
	Events(context.Context, *api.EventsOpts) error
}

type blockMonitor struct {
	conf             *config.Config
	client           BlockMonitorETH2RPC
	endpoint         config.Endpoint
	alertChannels    []alert.Alert
	lastSlot         phase0.Slot
	lastNewBlockTime time.Time
	log              logrus.Ext1FieldLogger
}

func NewBlockMonitor(conf *config.Config, client BlockMonitorETH2RPC, endpoint config.Endpoint, alertChannels []alert.Alert) monitor.Monitor {
	out := &blockMonitor{
		conf:          conf,
		client:        client,
		endpoint:      endpoint,
		alertChannels: alertChannels,
	}

	out.log = conf.Log.WithFields(logrus.Fields{
		"name":     out.Name(),
		"endpoint": endpoint.Name,
	})

	return out
}

func (bm blockMonitor) Name() string {
	return "consensus::BlockMonitor::" + bm.endpoint.Name
}

func (bm *blockMonitor) blockEventListen(ctx context.Context) error {
	return bm.client.Events(ctx, &api.EventsOpts{
		Topics: []string{"block"},
		BlockHandler: func(ctx context.Context, block *apiv1.BlockEvent) {
			bm.log.WithField("block", block).Debug("New block received")

			if block.Slot > bm.lastSlot {
				bm.lastSlot = block.Slot
			}

			bm.lastNewBlockTime = time.Now()

		},
	})
}

func (bm *blockMonitor) Run(ctx context.Context) {

	go func() {
		if err := bm.blockEventListen(ctx); err != nil {
			bm.log.WithError(err).Panic("block event listener exited with error")
			// TODO: we might want to either alert here, or reconnect
		}
	}()
	bm.lastNewBlockTime = time.Now()
	for {
		select {
		case <-ctx.Done():
			bm.log.Info("monitoring stopped")
			return
		default:
			if time.Since(bm.lastNewBlockTime) > bm.endpoint.NewBlockMaxDuration {
				alertErr := alert.RaiseAll(ctx, bm.log, bm.alertChannels, alert.Message{
					Message:  fmt.Sprintf("no new block for %d seconds, expected less than %d seconds", int64(time.Since(bm.lastNewBlockTime).Seconds()), int64(bm.endpoint.NewBlockMaxDuration.Seconds())),
					Severity: alert.Error,
					Name:     bm.endpoint.Name,
				})
				if alertErr != nil {
					bm.log.WithError(alertErr).Error("failed to raise alert")
				}
			} else {
				bm.log.WithFields(logrus.Fields{
					"slot": bm.lastSlot}).Info("Endpoint is healthy")
			}

			select {
			case <-time.After(bm.endpoint.PollDuration):
				continue
			case <-ctx.Done():
				bm.log.Info("monitoring stopped")
				return
			}
		}
	}
}
