package execution

import (
	"github.com/numbergroup/eth-monitor/pkg/alert"
	"github.com/numbergroup/eth-monitor/pkg/config"
	"github.com/numbergroup/eth-monitor/pkg/monitor"
	"github.com/numbergroup/eth-monitor/pkg/monitor/generic"
)

func NewPeerCountMonitor(conf *config.Config, alertChannels []alert.Alert, rpcClient generic.RPCPeerCount, endpoint config.Endpoint) (monitor.Monitor, error) {
	return generic.NewPeerCountMonitor(conf, alertChannels, rpcClient, endpoint, config.TypeExecution)
}
