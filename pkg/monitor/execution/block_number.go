package execution

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"

	"github.com/numbergroup/eth-monitor/pkg/alert"
	"github.com/numbergroup/eth-monitor/pkg/config"
	"github.com/numbergroup/eth-monitor/pkg/monitor"
)

type RPCBlockNumber interface {
	BlockNumber(ctx context.Context) (uint64, error)
}

type BlockNumberMonitor struct {
	alertChannels    []alert.Alert
	conf             *config.Config
	client           RPCBlockNumber
	endpoint         config.Endpoint
	lastBlockNumber  uint64
	lastNewBlockTime time.Time
	log              logrus.Ext1FieldLogger
}

func NewBlockNumberMonitor(conf *config.Config, alertChannels []alert.Alert, rpcClient RPCBlockNumber, endpoint config.Endpoint) (monitor.Monitor, error) {
	out := &BlockNumberMonitor{
		alertChannels: alertChannels,
		conf:          conf,
		client:        rpcClient,
		endpoint:      endpoint,
	}
	out.log = conf.Log.WithFields(logrus.Fields{
		"name":     out.Name(),
		"endpoint": endpoint.Name,
	})
	return out, nil
}

func (m *BlockNumberMonitor) checkNewBlock(ctx context.Context) error {
	blockNumber, err := m.client.BlockNumber(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get block number")
	}

	if blockNumber == m.lastBlockNumber {
		elapsedTime := time.Since(m.lastNewBlockTime)
		if elapsedTime > m.endpoint.NewBlockMaxDuration {
			return errors.Errorf("no new block for %s, expected less than %s", elapsedTime, m.endpoint.NewBlockMaxDuration.Seconds())
		}
	}

	if blockNumber < m.lastBlockNumber {
		m.lastBlockNumber = blockNumber
		m.lastNewBlockTime = time.Now()
		return errors.Errorf("block number decreased from %d to %d", m.lastBlockNumber, blockNumber)
	}
	m.lastBlockNumber = blockNumber
	m.lastNewBlockTime = time.Now()
	return nil
}

func (m *BlockNumberMonitor) Name() string {
	return "execution::BlockNumberMonitor::" + m.endpoint.Name
}

func (m *BlockNumberMonitor) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			m.log.Info("monitoring stopped")
			return
		default:
			if err := m.checkNewBlock(ctx); err != nil {
				m.log.WithError(err).Error("health check failed, raising alert")
				alertErr := alert.RaiseAll(ctx, m.log, m.alertChannels, alert.Message{
					Message:  err.Error(),
					Severity: alert.Error,
					Name:     m.endpoint.Name,
				})
				if alertErr != nil {
					m.conf.Log.WithError(alertErr).Error("failed to raise alert")
				}

			} else {
				m.log.WithFields(logrus.Fields{
					"block": m.lastBlockNumber}).Info("Endpoint is healthy")
			}
		}

		select {
		case <-time.After(m.endpoint.PollDuration):
			continue
		case <-ctx.Done():
			m.log.Info("monitoring stopped")
			return
		}
	}
}
