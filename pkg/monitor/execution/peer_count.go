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

// RPCPeerCount defines the minimal RPC surface needed for peer monitoring.
type RPCPeerCount interface {
	PeerCount(ctx context.Context) (uint64, error)
}

type PeerCountMonitor struct {
	alertChannels       []alert.Alert
	conf                *config.Config
	client              RPCPeerCount
	endpoint            config.Endpoint
	lastPeerCount       uint64
	hasEverBeenAboveMin bool // Tracks if peer count has ever been above minimum, to avoid false alerts on startup
	log                 logrus.Ext1FieldLogger
}

func NewPeerCountMonitor(conf *config.Config, alertChannels []alert.Alert, rpcClient RPCPeerCount, endpoint config.Endpoint) (monitor.Monitor, error) {
	return &PeerCountMonitor{
		alertChannels:       alertChannels,
		conf:                conf,
		client:              rpcClient,
		endpoint:            endpoint,
		log:                 conf.Log,
		hasEverBeenAboveMin: false,
	}, nil
}

func (m *PeerCountMonitor) checkPeerCount(ctx context.Context) error {
	pc, err := m.client.PeerCount(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get peer count")
	}
	m.lastPeerCount = pc

	if m.endpoint.MinPeers > 0 && int(pc) < m.endpoint.MinPeers && m.hasEverBeenAboveMin {
		return errors.Errorf("peer count %d below minimum %d", pc, m.endpoint.MinPeers)
	}
	if int(pc) > m.endpoint.MinPeers+2 {
		m.hasEverBeenAboveMin = true
	}

	return nil
}

func (m *PeerCountMonitor) Name() string { return "execution::PeerCountMonitor::" + m.endpoint.Name }

func (m *PeerCountMonitor) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			m.conf.Log.WithField("name", m.endpoint.Name).Info("monitoring stopped")
			return
		default:
			if err := m.checkPeerCount(ctx); err != nil {
				m.conf.Log.WithError(err).Error("health check failed, raising alert")
				alertErr := alert.RaiseAll(ctx, m.conf.Log, m.alertChannels, alert.Message{
					Message:  err.Error(),
					Severity: alert.Error,
					Name:     m.endpoint.Name,
				})
				if alertErr != nil {
					m.conf.Log.WithError(alertErr).Error("failed to raise alert")
				}
			} else {
				m.conf.Log.WithFields(logrus.Fields{
					"peers": m.lastPeerCount,
					"name":  m.endpoint.Name,
				}).Info("Endpoint is healthy")
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
