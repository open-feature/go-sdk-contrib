package manager

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/api"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
)

const dataCollectorMaxEventStoredDefault = 100000
const collectIntervalDefault = 2 * time.Minute

// DataCollectorManager is a manager for the GO Feature Flag data collector
type DataCollectorManager struct {
	mutex                       *sync.Mutex
	sendMutex                   *sync.Mutex
	goffAPI                     api.GoFeatureFlagAPI
	events                      []model.CollectableEvent
	dataCollectorMaxEventStored int64
	collectInterval             time.Duration
	logger                      *slog.Logger
	sendFailing                 bool

	ticker         *time.Ticker
	collectChannel chan bool
	flushChannel   chan struct{}
	goroutineDone  chan struct{}
}

// NewDataCollectorManager creates a new data collector manager; the optional logger defaults to slog.Default().
func NewDataCollectorManager(
	goffAPI api.GoFeatureFlagAPI,
	dataCollectorMaxEventStored int64,
	collectInterval time.Duration,
	logger ...*slog.Logger) DataCollectorManager {
	if dataCollectorMaxEventStored <= 0 {
		dataCollectorMaxEventStored = dataCollectorMaxEventStoredDefault
	}
	if collectInterval <= 0 {
		collectInterval = collectIntervalDefault
	}
	l := slog.Default()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return DataCollectorManager{
		logger:                      l,
		mutex:                       &sync.Mutex{},
		sendMutex:                   &sync.Mutex{},
		goffAPI:                     goffAPI,
		events:                      make([]model.CollectableEvent, 0),
		dataCollectorMaxEventStored: dataCollectorMaxEventStored,
		collectInterval:             collectInterval,
		collectChannel:              make(chan bool, 1),
		flushChannel:                make(chan struct{}, 1),
	}
}

func (d *DataCollectorManager) Start() {
	d.goroutineDone = make(chan struct{})
	d.ticker = time.NewTicker(d.collectInterval)
	tickerC := d.ticker.C
	go func() {
		defer close(d.goroutineDone)
		for {
			select {
			case <-d.collectChannel:
				return
			case <-tickerC:
				_ = d.SendData(context.Background())
			case <-d.flushChannel:
				_ = d.SendData(context.Background())
			}
		}
	}()
}

func (d *DataCollectorManager) Stop(ctx context.Context) {
	select {
	case d.collectChannel <- true:
	default:
	}
	if d.ticker != nil {
		d.ticker.Stop()
	}
	if d.goroutineDone != nil {
		<-d.goroutineDone
	}
	_ = d.SendData(ctx)
}

// SendData sends queued events to the data collector without holding the queue lock, re-queuing them on failure.
func (d *DataCollectorManager) SendData(ctx context.Context) error {
	d.sendMutex.Lock()
	defer d.sendMutex.Unlock()

	d.mutex.Lock()
	batch := d.events
	d.events = make([]model.CollectableEvent, 0)
	d.mutex.Unlock()

	if len(batch) == 0 {
		return nil
	}
	err := d.goffAPI.CollectData(ctx, batch)
	if err == nil {
		if d.sendFailing {
			d.sendFailing = false
			d.logger.Info("sending events to the relay-proxy data collector recovered")
		}
		return nil
	}

	d.mutex.Lock()
	d.events = d.trimOldest(append(batch, d.events...))
	queued := len(d.events)
	d.mutex.Unlock()
	if !d.sendFailing {
		d.sendFailing = true
		d.logger.Warn("failed to send events to the relay-proxy data collector, will retry; the oldest events are dropped once the queue is full",
			"error", err, "queued", queued, "maxEventStored", d.dataCollectorMaxEventStored)
	}
	return err
}

// AddEvent queues an event without doing I/O, waking the background sender when the queue becomes full and dropping the oldest on overflow.
func (d *DataCollectorManager) AddEvent(event model.CollectableEvent) error {
	d.mutex.Lock()
	before := int64(len(d.events))
	d.events = d.trimOldest(append(d.events, event))
	full := before < d.dataCollectorMaxEventStored && int64(len(d.events)) >= d.dataCollectorMaxEventStored
	d.mutex.Unlock()

	if full {
		select {
		case d.flushChannel <- struct{}{}:
		default:
		}
	}
	return nil
}

// trimOldest drops the oldest events so that at most dataCollectorMaxEventStored remain. Caller must hold d.mutex.
func (d *DataCollectorManager) trimOldest(events []model.CollectableEvent) []model.CollectableEvent {
	overflow := int64(len(events)) - d.dataCollectorMaxEventStored
	if overflow <= 0 {
		return events
	}
	return events[overflow:]
}
