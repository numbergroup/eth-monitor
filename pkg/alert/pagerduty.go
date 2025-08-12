package alert

import (
	"context"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/numbergroup/eth-monitor/pkg/config"
)

func NewPagerduty(conf *config.Config, endpoint config.Endpoint) Pagerduty {
	out := Pagerduty{
		Service: endpoint.Pagerduty.Service,
	}
	if endpoint.Pagerduty.RoutingKey != "" {
		out.RoutingKey = endpoint.Pagerduty.RoutingKey
	} else {
		out.RoutingKey = conf.Pagerduty.RoutingKey
	}

	return out
}

type Pagerduty struct {
	RoutingKey string
	Service    string
}

// TODO: Prevent duplicate alerts by checking if an alert is already active
func (p Pagerduty) Raise(ctx context.Context, msg Message) error {

	payload := &pagerduty.V2Payload{
		Summary:   msg.Message,
		Severity:  string(msg.Severity),
		Component: msg.Name,
		Source:    p.Service,
		Details:   msg.Metadata,
	}

	_, err := pagerduty.ManageEventWithContext(ctx, pagerduty.V2Event{
		RoutingKey: p.RoutingKey,
		Action:     "trigger",
		Payload:    payload,
	})
	return err
}
