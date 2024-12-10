package process

import (
	context "context"

	"github.com/open-feature/flagd/core/pkg/sync"
)

// Fake implementation of sync.ISync. Does not conform to the contract because it does not send any events to the DataSync.
// Only used for unit tests.
type DoNothingCustomSyncProvider struct {
}

func (fps DoNothingCustomSyncProvider) Init(ctx context.Context) error {
	return nil
}

func (fps DoNothingCustomSyncProvider) IsReady() bool {
	return true
}

func (fps DoNothingCustomSyncProvider) Sync(ctx context.Context, dataSync chan<- sync.DataSync) error {
	return nil
}

func (fps DoNothingCustomSyncProvider) ReSync(ctx context.Context, dataSync chan<- sync.DataSync) error {
	return nil
}

// Returns an implementation of sync.ISync interface that does nothing at all.
// The returned implementation does not conform to the sync.DataSync contract.
// This is useful only for unit tests.
func NewDoNothingCustomSyncProvider() DoNothingCustomSyncProvider {
	return DoNothingCustomSyncProvider{}
}

var _ sync.ISync = &DoNothingCustomSyncProvider{}
