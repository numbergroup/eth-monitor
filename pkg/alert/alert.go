package alert

import "context"

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
