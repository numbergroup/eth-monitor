package alert

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/numbergroup/eth-monitor/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

func NewSlack(conf *config.Config, endpoint config.Endpoint) Slack {
	return Slack{
		conf:     conf,
		log:      conf.Log,
		endpoint: endpoint,
	}
}

type Slack struct {
	conf     *config.Config
	endpoint config.Endpoint
	log      logrus.Ext1FieldLogger
}

func (s Slack) Raise(ctx context.Context, msg Message) error {
	// Handle Slack, first try the endpoint's Slack configuration, then the global Slack configuration
	// Webhook is prioritized, but if it's empty, we can use the channel and token

	slackMsg := &slack.WebhookMessage{
		Text: s.formatMessage(msg),
		Attachments: []slack.Attachment{
			{
				Color:  s.severityColor(msg.Severity),
				Fields: s.buildMetadataFields(msg),
			},
		},
	}

	// Priority: Endpoint Webhook -> Endpoint Channel/Token -> Global Webhook -> Global Channel/Token

	// 1. Try endpoint webhook URL first
	if len(s.endpoint.Slack.WebhookURL) != 0 {
		err := slack.PostWebhookContext(ctx, s.endpoint.Slack.WebhookURL, slackMsg)
		if err != nil {
			s.log.WithError(err).WithField("endpoint", msg.Name).Error("failed to send Slack alert via endpoint webhook")
		}
		return err
	}

	// 2. Try endpoint channel/token if available
	if len(s.endpoint.Slack.Channel) != 0 && len(s.endpoint.Slack.Token) != 0 {
		api := slack.New(s.endpoint.Slack.Token)
		_, _, err := api.PostMessageContext(ctx, s.endpoint.Slack.Channel,
			slack.MsgOptionText(s.formatMessage(msg), false),
			slack.MsgOptionAttachments(slack.Attachment{
				Color:  s.severityColor(msg.Severity),
				Fields: s.buildMetadataFields(msg),
			}),
		)
		if err != nil {
			s.log.WithError(err).WithField("endpoint", msg.Name).Error("failed to send Slack alert via endpoint channel")
		}
		return err
	}

	// 3. Try global webhook URL
	if len(s.conf.Slack.WebhookURL) != 0 {
		err := slack.PostWebhookContext(ctx, s.conf.Slack.WebhookURL, slackMsg)
		if err != nil {
			s.log.WithError(err).WithField("endpoint", msg.Name).Error("failed to send Slack alert via global webhook")
		}
		return err
	}

	// 4. Try global channel/token as last resort
	if len(s.conf.Slack.Channel) != 0 && len(s.conf.Slack.Token) != 0 {
		api := slack.New(s.conf.Slack.Token)
		_, _, err := api.PostMessageContext(ctx, s.conf.Slack.Channel,
			slack.MsgOptionText(s.formatMessage(msg), false),
			slack.MsgOptionAttachments(slack.Attachment{
				Color:  s.severityColor(msg.Severity),
				Fields: s.buildMetadataFields(msg),
			}),
		)
		if err != nil {
			s.log.WithError(err).WithField("endpoint", msg.Name).Error("failed to send Slack alert via global channel")
		}
		return err
	}

	return errors.New("no valid Slack configuration found for alerting")
}

func (s Slack) formatMessage(msg Message) string {
	return fmt.Sprintf("[%s] %s: %s", strings.ToUpper(string(msg.Severity)), msg.Name, msg.Message)
}

func (s Slack) severityColor(severity Severity) string {
	switch severity {
	case Error:
		return "danger"
	default:
		return "warning"
	}
}

func (s Slack) buildMetadataFields(msg Message) []slack.AttachmentField {
	var fields []slack.AttachmentField

	for key, value := range msg.Metadata {
		fields = append(fields, slack.AttachmentField{
			Title: key,
			Value: fmt.Sprintf("%v", value),
			Short: true,
		})
	}

	return fields
}
