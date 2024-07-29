package controller

import (
	"fmt"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/provider_v2/model"
	"sync"
	"time"
)

// DataCollectorManager is a manager for the GO Feature Flag data collector
type DataCollectorManager struct {
	mutex   *sync.RWMutex
	goffAPI GoFeatureFlagAPI
	events  []model.FeatureEvent
	maxItem int64

	ticker         *time.Ticker
	collectChannel chan bool
}

// NewDataCollectorManager creates a new data collector manager
func NewDataCollectorManager(
	goffAPI GoFeatureFlagAPI,
	maxItem int64,
	collectInterval time.Duration) *DataCollectorManager {
	return &DataCollectorManager{
		mutex:          &sync.RWMutex{},
		goffAPI:        goffAPI,
		events:         make([]model.FeatureEvent, 0),
		maxItem:        maxItem,
		ticker:         time.NewTicker(collectInterval),
		collectChannel: make(chan bool),
	}
}

func (d *DataCollectorManager) Start() {
	// Start the goroutine
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

// SendData sends the data to the data collector
func (d *DataCollectorManager) SendData() error {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if len(d.events) <= 0 {
		return nil
	}

	copySend := make([]model.FeatureEvent, len(d.events))
	copy(copySend, d.events)
	err := d.goffAPI.CollectData(copySend)
	if err != nil {
		return err
	}
	d.events = make([]model.FeatureEvent, 0)
	return nil
}

// AddEvent adds an event to the data collector manager
// If the number of events in the queue is greater than the maxItem, the event will be skipped
func (d *DataCollectorManager) AddEvent(event model.FeatureEvent) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if nbItem := int64(len(d.events)); nbItem >= d.maxItem {
		return fmt.Errorf("too many events in the queue, this event will be skipped: %d", nbItem)
	}

	d.events = append(d.events, event)
	return nil
}
