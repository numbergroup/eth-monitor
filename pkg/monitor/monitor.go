package monitor

import "context"

type Monitor interface {
	Run(ctx context.Context)
	Name() string
}
