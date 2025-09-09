package execution

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"

	"github.com/numbergroup/eth-monitor/pkg/alert"
	"github.com/numbergroup/eth-monitor/pkg/config"
)

type RPCBlockNumber interface {
	BlockNumber(ctx context.Context) (uint64, error)
}

type BlockNumberMonitor struct {
	alertChannels    []alert.Alert
	conf             *config.Config
	client           ETHRPC
	endpoint         config.Endpoint
	lastBlockNumber  uint64
	lastNewBlockTime time.Time
	log              logrus.Ext1FieldLogger
}

func NewBlockNumberMonitor(conf *config.Config, alertChannels []alert.Alert, rpcClient RPCBlockNumber, endpoint config.Endpoint) (Monitor, error) {
	return &BlockNumberMonitor{
		alertChannels: alertChannels,
		conf:          conf,
		client:        rpcClient,
		endpoint:      endpoint,
		log:           conf.Log,
	}, nil
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
	return "BlockNumberMonitor::" + m.endpoint.Name
}

func (m *BlockNumberMonitor) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			m.conf.Log.WithField("name", m.endpoint.Name).Info("monitoring stopped")
			return
		default:
			if err := m.checkNewBlock(ctx); err != nil {
				m.conf.Log.WithError(err).Error("health check failed, raising alert")
				for _, alertChannel := range m.alertChannels {
					alertErr := alertChannel.Raise(ctx, alert.Message{
						Message:  err.Error(),
						Severity: alert.Error,
						Name:     m.endpoint.Name,
					})
					if alertErr != nil {
						m.conf.Log.WithError(alertErr).Error("failed to raise alert")
					}
				}

			} else {
				m.conf.Log.WithFields(logrus.Fields{
					"block": m.lastBlockNumber,
					"name":  m.endpoint.Name}).Info("Endpoint is healthy")
			}
		}

		select {
		case <-time.After(m.endpoint.PollDuration):
			continue
		case <-ctx.Done():
			m.conf.Log.WithField("name", m.endpoint.Name).Info("monitoring stopped")
			return
		}
	}
}
