package config

import (
	"context"
	"os"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/goccy/go-yaml"
	"github.com/sirupsen/logrus"
)

type Pagerduty struct {
	RoutingKey string `yaml:"routing_key"`
	Group      string `yaml:"group"`
}

func (p Pagerduty) Empty() bool {
	return p.RoutingKey == ""
}

func (p Pagerduty) RaiseAlert(ctx context.Context, name, issue string) error {
	if p.Empty() {
		return nil
	}
	_, err := pagerduty.ManageEventWithContext(ctx, pagerduty.V2Event{
		RoutingKey: p.RoutingKey,
		Action:     "trigger",
		Payload: &pagerduty.V2Payload{
			Summary:   issue,
			Severity:  "error",
			Component: name,
			Source:    name,
			Group:     p.Group,
		},
	})
	return err
}

type Slack struct {
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
	Token      string `yaml:"token"`
}

func (s Slack) Empty() bool {
	return len(s.WebhookURL) == 0 && len(s.Channel) == 0 && len(s.Token) == 0
}

type Config struct {
	Endpoints []Endpoint `yaml:"endpoints"`
	Pagerduty Pagerduty  `yaml:"pagerduty"`
	Slack     Slack      `yaml:"slack"`
	Verbosity string     `yaml:"verbosity"`

	Log logrus.Ext1FieldLogger `yaml:"-"` // Log field is not serialized to YAML, used for logging
}

func LoadConfig(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	conf := &Config{}
	err = yaml.Unmarshal(data, conf)
	if err != nil {
		return nil, err
	}
	logger := logrus.New()
	lvl, err := logrus.ParseLevel(conf.Verbosity)
	if err != nil {
		logger.SetLevel(logrus.InfoLevel)
	} else {
		logger.SetLevel(lvl)
	}

	conf.Log = logger
	return conf, err
}
