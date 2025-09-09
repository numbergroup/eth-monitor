package alert

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Message struct {
	Message  string
	Severity Severity
	Name     string
	Metadata map[string]any
}

type Alert interface {
	Raise(ctx context.Context, msg Message) error
}

type Severity string

const (
	Error Severity = "error"
)

func RaiseAll(ctx context.Context, logger logrus.Ext1FieldLogger, alertChannels []Alert, msg Message) error {
	var err error
	for _, alertChannel := range alertChannels {
		alertErr := alertChannel.Raise(ctx, msg)
		if alertErr != nil {
			logger.WithError(alertErr).Error("failed to raise alert")
		}
	}
	return err
}
