package flagd

import "context"

type Cache interface {
	Set(ctx context.Context, flagKey string, value interface{}) error
	Get(ctx context.Context, flagKey string) (interface{}, error)
	Delete(ctx context.Context, flagKey string) error
	DeleteAll(ctx context.Context) error
}
