package alert

import (
	"context"
	"errors"

	"github.com/numbergroup/eth-monitor/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

func NewSlack(conf *config.Config) Slack {
	return Slack{
		Log:        conf.Log,
		WebhookURL: conf.Slack.WebhookURL,
		Channel:    conf.Slack.Channel,
		Token:      conf.Slack.Token,
	}
}

type Slack struct {
	Log        logrus.Ext1FieldLogger
	WebhookURL string
	Channel    string
	Token      string
}

// TODO: Include more details, like metadata, severity, etc.
func (s Slack) Raise(ctx context.Context, msg Message) error {
	// Handle Slack, first try the endpoint's Slack configuration, then the global Slack configuration
	// Webhook is prioritized, but if it's empty, we can use the channel and token

	slackMsg := &slack.WebhookMessage{
		Text: msg.Message}

	// Priority Enpoint Slack Webhook URL -> Endpoint Slack Channel/Token -> Global Slack Webhook URL -> Global Slack Channel/Token
	if len(s.WebhookURL) != 0 {
		err := slack.PostWebhookContext(ctx, s.WebhookURL, slackMsg)
		if err != nil {
			s.Log.WithError(err).WithField("endpoint", msg.Name).Error("failed to send Slack alert via webhook")
		}
		return err
	}
	if len(s.Channel) == 0 && len(s.Token) == 0 && len(s.WebhookURL) != 0 {
		err := slack.PostWebhookContext(ctx, s.WebhookURL, slackMsg)
		if err != nil {
			s.Log.WithError(err).WithField("endpoint", msg.Name).Error("failed to send Slack alert via webhook")
		}
		return err
	}

	token := s.Token
	if len(token) == 0 {
		token = s.Token
		if len(token) == 0 {
			return errors.New("no Slack token provided for alerting")
		}
	}

	channel := s.Channel
	if len(channel) == 0 {
		channel = s.Channel
		if len(channel) == 0 {
			return errors.New("no Slack channel provided for alerting")
		}
	}
	api := slack.New(s.Token)
	_, _, err := api.PostMessageContext(ctx, channel, slack.MsgOptionText(msg.Message, false))
	if err != nil {
		s.Log.WithError(err).WithField("endpoint", msg.Name).Error("failed to send Slack alert via channel")
	}
	return err
}
