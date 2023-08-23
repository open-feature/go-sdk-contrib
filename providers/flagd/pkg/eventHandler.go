package flagd

import (
	schemaV1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	"context"
	"errors"
	"github.com/go-logr/logr"
	flagdService "github.com/open-feature/flagd/core/pkg/service"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
)

// eventHandler abstracts the event handling of flagd
type eventHandler struct {
	cache     *cacheService
	errChan   chan error
	eventChan chan *schemaV1.EventStreamResponse
	isReady   chan struct{}
	logger    logr.Logger
}

func (h *eventHandler) handle(ctx context.Context) error {
	for {
		select {
		case event, ok := <-h.eventChan:
			if !ok {
				return errors.New("event channel closed")
			}

			switch event.Type {
			case string(flagdService.ConfigurationChange):
				if err := h.handleConfigurationChangeEvent(event); err != nil {
					// Purge the cache if we fail to handle the configuration change event
					(*h.cache).getCache().Purge()
					h.logger.V(logger.Warn).Info("handle configuration change event", "err", err)
				}
			case string(flagdService.ProviderReady): // signals that a new connection has been made
				h.handleProviderReadyEvent()
			}
		case err := <-h.errChan:
			// purge for error
			(*h.cache).getCache().Purge()
			return err
		case <-ctx.Done():
			h.logger.V(logger.Info).Info("stop event handling with context done")
			return nil
		}
	}
}

func (h *eventHandler) handleConfigurationChangeEvent(event *schemaV1.EventStreamResponse) error {
	if event.Data == nil {
		return errors.New("no data in event")
	}

	flagsVal, ok := event.Data.AsMap()["flags"]
	if !ok {
		return errors.New("no flags field in event data")
	}

	flags, ok := flagsVal.(map[string]interface{})
	if !ok {
		return errors.New("flags isn't a map")
	}

	for flagKey := range flags {
		(*h.cache).getCache().Remove(flagKey)
	}

	return nil
}

// todo - this needs to migrate to OF eventing
func (h *eventHandler) handleProviderReadyEvent() {
	select {
	case <-h.isReady:
		// avoids panic from closing already closed channel
	default:
		close(h.isReady)
	}
}
