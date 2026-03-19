package manager

import (
	"context"
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
	goffAPI                     api.GoFeatureFlagAPI
	events                      []model.CollectableEvent
	dataCollectorMaxEventStored int64
	collectInterval             time.Duration

	ticker         *time.Ticker
	collectChannel chan bool
	goroutineDone  chan struct{}
}

// NewDataCollectorManager creates a new data collector manager
func NewDataCollectorManager(
	goffAPI api.GoFeatureFlagAPI,
	dataCollectorMaxEventStored int64,
	collectInterval time.Duration) DataCollectorManager {
	if dataCollectorMaxEventStored <= 0 {
		dataCollectorMaxEventStored = dataCollectorMaxEventStoredDefault
	}
	if collectInterval <= 0 {
		collectInterval = collectIntervalDefault
	}
	return DataCollectorManager{
		mutex:                       &sync.Mutex{},
		goffAPI:                     goffAPI,
		events:                      make([]model.CollectableEvent, 0),
		dataCollectorMaxEventStored: dataCollectorMaxEventStored,
		collectInterval:             collectInterval,
		collectChannel:              make(chan bool, 1),
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

// sendDataLocked flushes events to the API. Caller must hold d.mutex.
func (d *DataCollectorManager) sendDataLocked(ctx context.Context) error {
	if len(d.events) == 0 {
		return nil
	}
	copySend := make([]model.CollectableEvent, len(d.events))
	copy(copySend, d.events)
	if err := d.goffAPI.CollectData(ctx, copySend); err != nil {
		return err
	}
	d.events = make([]model.CollectableEvent, 0)
	return nil
}

// SendData sends the data to the data collector
func (d *DataCollectorManager) SendData(ctx context.Context) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.sendDataLocked(ctx)
}

// AddEvent adds an event (FeatureEvent or TrackingEvent) to the data collector manager.
// If the queue is full, we flush first. If the flush fails, the new event is not added and the error is returned.
func (d *DataCollectorManager) AddEvent(event model.CollectableEvent) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if int64(len(d.events)) >= d.dataCollectorMaxEventStored {
		if err := d.sendDataLocked(context.Background()); err != nil {
			return err
		}
	}

	d.events = append(d.events, event)
	return nil
}
