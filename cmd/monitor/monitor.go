package main

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/numbergroup/eth-monitor/pkg/config"
)

type monitor struct {
	endpoint config.Endpoint
	conf     *config.Config
	client   *ethclient.Client

	lastBlockNumber  uint64
	lastNewBlockTime time.Time
}

func NewMonitor(ctx context.Context, conf *config.Config, endpoint config.Endpoint) (*monitor, error) {
	client, err := ethclient.DialContext(ctx, endpoint.URL)
	if err != nil {
		return nil, err
	}
	return &monitor{
		endpoint: endpoint,
		conf:     conf,
		client:   client,
	}, nil
}

func (m *monitor) checkNewBlock(ctx context.Context) error {
	blockNumber, err := m.client.BlockNumber(ctx)
	if err != nil {
		return err
	}

	if blockNumber == m.lastBlockNumber {
		elapsedTime := time.Since(m.lastNewBlockTime)
		if elapsedTime > m.endpoint.NewBlockMaxDuration {
			return errors.Errorf("no new block for %s, expected less than %s", elapsedTime, m.endpoint.NewBlockMaxDuration.Seconds())
		}
	}

	if blockNumber < m.lastBlockNumber {
		return errors.Errorf("block number decreased from %d to %d", m.lastBlockNumber, blockNumber)
	}
	m.lastBlockNumber = blockNumber
	m.lastNewBlockTime = time.Now()
	return nil
}

func (m *monitor) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			m.conf.Log.WithField("name", m.endpoint.Name).Info("monitoring stopped")
			return
		default:
			if err := m.checkNewBlock(ctx); err != nil {
				m.conf.Log.WithError(err).Error("health check failed, raising alert")
				alertErr := m.endpoint.RaiseAlert(ctx, m.conf, err.Error())
				if alertErr != nil {
					m.conf.Log.WithError(alertErr).Error("failed to raise alert")
				}

			} else {
				m.conf.Log.WithField("name", m.endpoint.Name).Info("Endpoint is healthy")
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
