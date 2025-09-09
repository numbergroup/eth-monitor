package consensus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/numbergroup/eth-monitor/pkg/alert"
	"github.com/numbergroup/eth-monitor/pkg/config"
	"github.com/numbergroup/eth-monitor/pkg/monitor"
	"github.com/numbergroup/eth-monitor/pkg/monitor/generic"
)

type peerCount struct {
	endpoint config.Endpoint
	client   *http.Client
}

func (pc *peerCount) PeerCount(ctx context.Context) (uint64, error) {
	reqURL, err := url.JoinPath(pc.endpoint.URL, "/eth/v1/node/peer_count")
	if err != nil {
		return 0, errors.Wrapf(err, "failed to create peer count URL for endpoint %s", pc.endpoint.Name)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to create peer count request for endpoint %s", pc.endpoint.Name)
	}

	resp, err := pc.client.Do(req)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to perform peer count request for endpoint %s", pc.endpoint.Name)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, errors.Errorf("unexpected status code %d from peer count request for endpoint %s", resp.StatusCode, pc.endpoint.Name)
	}

	var result struct {
		Data map[string]string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, errors.Wrapf(err, "failed to decode peer count response for endpoint %s", pc.endpoint.Name)
	}

	peerCountStr, ok := result.Data["connected"]
	if !ok {
		return 0, errors.Errorf("missing 'connected' field in peer count response for endpoint %s", pc.endpoint.Name)
	}

	peerCount, err := strconv.ParseUint(peerCountStr, 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse peer count for endpoint %s", pc.endpoint.Name)
	}

	return peerCount, nil
}

func NewPeerCountClient(endpoint config.Endpoint) generic.RPCPeerCount {
	return &peerCount{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func NewPeerCountMonitor(conf *config.Config, alertChannels []alert.Alert, endpoint config.Endpoint) (monitor.Monitor, error) {
	return generic.NewPeerCountMonitor(conf, alertChannels, NewPeerCountClient(endpoint), endpoint, config.TypeConsensus)
}
