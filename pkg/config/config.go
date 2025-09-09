package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-yaml"
	"github.com/sirupsen/logrus"
)

const (
	TypeExecution          = "execution"
	TypeConsensus          = "consensus"
	PeerStartThreshold int = 2
)

type Pagerduty struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	RoutingKey string `yaml:"routing_key" json:"routing_key"`
	Service    string `yaml:"service" json:"service"`
}

func (p Pagerduty) Empty() bool {
	return p.RoutingKey == "" || !p.Enabled
}

type Slack struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	WebhookURL string `yaml:"webhook_url" json:"webhook_url"`
	Channel    string `yaml:"channel" json:"channel"`
	Token      string `yaml:"token" json:"token"`
}

func (s Slack) Empty() bool {
	return !s.Enabled || (len(s.WebhookURL) == 0 && len(s.Channel) == 0 && len(s.Token) == 0)
}

type Config struct {
	Endpoints  []Endpoint    `yaml:"endpoints" json:"endpoints"`
	RPCTimeout time.Duration `yaml:"rpc_timeout" json:"rpc_timeout"`
	Pagerduty  Pagerduty     `yaml:"pagerduty" json:"pagerduty"`
	Slack      Slack         `yaml:"slack" json:"slack"`
	Verbosity  string        `yaml:"verbosity" json:"verbosity"`

	Log logrus.Ext1FieldLogger `yaml:"-" json:"-"` // Log field is not serialized to YAML, used for logging
}

func LoadConfigFromFile(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return LoadConfig(data)
}

func LoadConfig(data []byte) (*Config, error) {
	conf := &Config{}
	err := yaml.Unmarshal(data, conf)
	if err != nil {
		err = json.Unmarshal(data, conf)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse configuration")
		}
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
	if conf.RPCTimeout == 0 {
		conf.RPCTimeout = 10 * time.Second
	}
	return conf, nil
}

type Endpoint struct {
	Name                string        `yaml:"name" json:"name"`
	URL                 string        `yaml:"url" json:"url"`
	Type                string        `yaml:"type" json:"type"`
	NewBlockMaxDuration time.Duration `yaml:"new_block_max_duration" json:"new_block_max_duration"`
	MinPeers            int           `yaml:"min_peers" json:"min_peers"`
	Pagerduty           Pagerduty     `yaml:"pagerduty" json:"pagerduty"`
	Slack               Slack         `yaml:"slack" json:"slack"`
	PollDuration        time.Duration `yaml:"poll_duration" json:"poll_duration"`
}

func (e Endpoint) Validate() error {
	if len(e.Name) == 0 {
		return errors.New("endpoint name is required")
	}
	if len(e.URL) == 0 {
		return errors.New("endpoint URL is required")
	}

	switch e.Type {
	case TypeExecution:
	case TypeConsensus:
	default:
		return errors.Errorf("invalid endpoint type: %s", e.Type)
	}

	return nil
}
