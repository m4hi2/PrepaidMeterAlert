package datasources

import "context"

type DataFetcher interface {
	GetBalance(context.Context, Identifier) (Balance, error)
	Name() string
}
