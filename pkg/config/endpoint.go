package config

import (
	"context"
	"errors"
	"time"

	"github.com/slack-go/slack"
)

type Endpoint struct {
	Name                string        `yaml:"name"`
	URL                 string        `yaml:"url"`
	NewBlockMaxDuration time.Duration `yaml:"new_block_max_duration"`
	Pagerduty           Pagerduty     `yaml:"pagerduty"`
	Slack               Slack         `yaml:"slack"`
	PollDuration        time.Duration `yaml:"poll_duration"`
}

func (e Endpoint) pagerdutyAlert(ctx context.Context, conf *Config, issue string) error {
	// If the endpoint has its own PagerDuty configuration, use it
	if !e.Pagerduty.Empty() {
		return e.Pagerduty.RaiseAlert(ctx, e.Name, issue)
	}
	// Otherwise, use the global PagerDuty configuration
	if !conf.Pagerduty.Empty() {
		return conf.Pagerduty.RaiseAlert(ctx, e.Name, issue)
	}
	return nil
}

func (e Endpoint) slackAlert(ctx context.Context, conf *Config, issue string) error {
	if !e.Slack.Enabled && !conf.Slack.Enabled {
		return nil // No Slack alerting configured
	}
	// Handle Slack, first try the endpoint's Slack configuration, then the global Slack configuration
	// Webhook is prioritized, but if it's empty, we can use the channel and token

	// Priority Enpoint Slack Webhook URL -> Endpoint Slack Channel/Token -> Global Slack Webhook URL -> Global Slack Channel/Token
	if e.Slack.Enabled && len(e.Slack.WebhookURL) != 0 {
		err := slack.PostWebhookContext(ctx, e.Slack.WebhookURL, &slack.WebhookMessage{
			Text: issue})
		if err != nil {
			conf.Log.WithError(err).WithField("endpoint", e.Name).Error("failed to send Slack alert via webhook")
		}
		return err
	}
	if len(e.Slack.Channel) == 0 && len(e.Slack.Token) == 0 && len(conf.Slack.WebhookURL) != 0 && conf.Slack.Enabled {
		err := slack.PostWebhookContext(ctx, conf.Slack.WebhookURL, &slack.WebhookMessage{
			Text: issue})
		if err != nil {
			conf.Log.WithError(err).WithField("endpoint", e.Name).Error("failed to send Slack alert via webhook")
		}
		return err
	}

	token := e.Slack.Token
	if len(token) == 0 || !e.Slack.Enabled {
		token = conf.Slack.Token
		if len(token) == 0 {
			return errors.New("no Slack token provided for alerting")
		}
	}

	channel := e.Slack.Channel
	if len(channel) == 0 || !e.Slack.Enabled {
		channel = conf.Slack.Channel
		if len(channel) == 0 {
			return errors.New("no Slack channel provided for alerting")
		}
	}
	api := slack.New(e.Slack.Token)
	_, _, err := api.PostMessageContext(ctx, channel, slack.MsgOptionText(issue, false))
	if err != nil {
		conf.Log.WithError(err).WithField("endpoint", e.Name).Error("failed to send Slack alert via channel")
	}
	return err
}

func (e Endpoint) RaiseAlert(ctx context.Context, conf *Config, issue string) error {
	err := e.pagerdutyAlert(ctx, conf, issue)
	if err != nil {
		conf.Log.WithError(err).WithField("endpoint", e.Name).Error("failed to raise alert via PagerDuty")
	}

	err = e.slackAlert(ctx, conf, issue)
	if err != nil {
		conf.Log.WithError(err).WithField("endpoint", e.Name).Error("failed to raise alert via Slack")
	}
	return err
}
