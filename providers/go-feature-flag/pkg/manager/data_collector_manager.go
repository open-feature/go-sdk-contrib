package manager

import (
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

	ticker         *time.Ticker
	collectChannel chan bool
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
		ticker:                      time.NewTicker(collectInterval),
		collectChannel:              make(chan bool),
	}
}

func (d *DataCollectorManager) Start() {
	go func() {
		for {
			select {
			case <-d.collectChannel:
				return
			case <-d.ticker.C:
				_ = d.SendData()
			}
		}
	}()
}

func (d *DataCollectorManager) Stop() {
	d.collectChannel <- true
	d.ticker.Stop()
}

// sendDataLocked flushes events to the API. Caller must hold d.mutex.
func (d *DataCollectorManager) sendDataLocked() error {
	if len(d.events) == 0 {
		return nil
	}
	copySend := make([]model.CollectableEvent, len(d.events))
	copy(copySend, d.events)
	if err := d.goffAPI.CollectData(copySend); err != nil {
		return err
	}
	d.events = make([]model.CollectableEvent, 0)
	return nil
}

// SendData sends the data to the data collector
func (d *DataCollectorManager) SendData() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.sendDataLocked()
}

// AddEvent adds an event (FeatureEvent or TrackingEvent) to the data collector manager.
// If the queue is full, it flushes existing events first. If the flush fails and the queue
// is still full, the event is dropped and an error is returned.
func (d *DataCollectorManager) AddEvent(event model.CollectableEvent) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if int64(len(d.events)) >= d.dataCollectorMaxEventStored {
		if err := d.sendDataLocked(); err != nil {
			return err
		}
	}

	d.events = append(d.events, event)
	return nil
}
