package execution

import "context"

type ETHRPC interface {
	BlockNumber(ctx context.Context) (uint64, error)
}

type Monitor interface {
	Run(ctx context.Context)
	Name() string
}
