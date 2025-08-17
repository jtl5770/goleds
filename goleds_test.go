package main

import (
	"os"
	"sync"
	"testing"
	"time"

	c "lautenbacher.net/goleds/config"
	pl "lautenbacher.net/goleds/platform"
	p "lautenbacher.net/goleds/producer"
	u "lautenbacher.net/goleds/util"
)

type MockPlatform struct {
	pl.Platform
	ledWriter    chan []p.Led
	sensorEvents chan *u.Trigger
	sensors      map[string]c.SensorCfg
	lastLeds     [][]p.Led
	mu           sync.Mutex
	stopChan     chan struct{}
}

func (m *MockPlatform) GetLedWriter() chan<- []p.Led {
	return m.ledWriter
}

func (m *MockPlatform) GetSensorEvents() <-chan *u.Trigger {
	return m.sensorEvents
}

func (m *MockPlatform) GetSensorLedIndices() map[string]int {
	indices := make(map[string]int)
	for uid, cfg := range m.sensors {
		indices[uid] = cfg.LedIndex
	}
	return indices
}

func (m *MockPlatform) Start(pool *sync.Pool) error {
	go func() {
		for {
			select {
			case leds := <-m.ledWriter:
				m.mu.Lock()
				// Make a copy of the slice to avoid data races
				ledsCopy := make([]p.Led, len(leds))
				copy(ledsCopy, leds)
				m.lastLeds = append(m.lastLeds, ledsCopy)
				m.mu.Unlock()
			case <-m.stopChan:
				return
			}
		}
	}()
	return nil
}

func (m *MockPlatform) Stop() {
	close(m.stopChan)
}

func (m *MockPlatform) GetForceUpdateDelay() time.Duration {
	return 1 * time.Second
}

func (m *MockPlatform) GetLedsTotal() int {
	return 10
}

func (m *MockPlatform) Ready() <-chan bool {
	readyChan := make(chan bool)
	close(readyChan)
	return readyChan
}

func (m *MockPlatform) GetLastLeds() [][]p.Led {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to avoid race conditions
	ret := make([][]p.Led, len(m.lastLeds))
	for i, leds := range m.lastLeds {
		ret[i] = make([]p.Led, len(leds))
		copy(ret[i], leds)
	}
	return ret
}

func (m *MockPlatform) ClearLastLeds() {
	m.mu.Lock()
	m.lastLeds = nil
	m.mu.Unlock()
}

func NewMockPlatform() *MockPlatform {
	return &MockPlatform{
		ledWriter:    make(chan []p.Led, 1),
		sensorEvents: make(chan *u.Trigger),
		sensors:      make(map[string]c.SensorCfg),
		lastLeds:     make([][]p.Led, 0),
		stopChan:     make(chan struct{}),
	}
}

type MockLedProducer struct {
	*p.AbstractProducer
	uid       string
	isRunning bool
	leds      []p.Led
}

func (m *MockLedProducer) Start() {
	m.isRunning = true
}

func (m *MockLedProducer) SendTrigger(trigger *u.Trigger) {
	// Simulate the real producer starting when it receives a trigger.
	m.isRunning = true
}

func (m *MockLedProducer) TryStop() (bool, error) {
	wasRunning := m.isRunning
	m.isRunning = false
	return wasRunning, nil
}

func (m *MockLedProducer) GetIsRunning() bool {
	return m.isRunning
}

func (m *MockLedProducer) GetLeds(buffer []p.Led) {
	copy(buffer, m.leds)
}

func (m *MockLedProducer) Exit() {
	// do nothing
}

func (m *MockLedProducer) GetUID() string {
	return m.uid
}

func NewMockLedProducer(uid string) *MockLedProducer {
	return &MockLedProducer{
		uid:       uid,
		isRunning: false,
		leds:      make([]p.Led, 0),
	}
}

func TestStateManager(t *testing.T) {
	ossignal := make(chan os.Signal, 1)
	app := NewApp(ossignal)
	app.ledproducers = make(map[string]p.LedProducer)

	mockPlatform := NewMockPlatform()
	app.platform = mockPlatform

	mockProducer := NewMockLedProducer("test")
	app.ledproducers["test"] = mockProducer

	app.stopsignal = make(chan struct{})
	app.shutdownWg.Add(1)
	go app.stateManager()
	t.Cleanup(func() {
		close(app.stopsignal)
		app.shutdownWg.Wait()
	})

	// test normal trigger
	mockPlatform.sensorEvents <- u.NewTrigger("test", 10, time.Now())
	time.Sleep(100 * time.Millisecond)
	if !mockProducer.GetIsRunning() {
		t.Error("Expected producer to be running")
	}
	mockProducer.TryStop()
}

func TestCombineAndUpdateDisplay(t *testing.T) {
	ossignal := make(chan os.Signal, 1)
	app := NewApp(ossignal)
	app.ledproducers = make(map[string]p.LedProducer)

	mockPlatform := NewMockPlatform()
	app.platform = mockPlatform
	// Start the mock platform to begin capturing LED data
	mockPlatform.Start(nil)
	t.Cleanup(mockPlatform.Stop)

	mockPlatform.sensors["sensor"] = c.SensorCfg{LedIndex: 0, SpiMultiplex: "", AdcChannel: 0, TriggerValue: 0}

	mockSensorProducer := NewMockLedProducer("sensor")
	mockMultiBlobProducer := NewMockLedProducer(MULTI_BLOB_UID)
	app.ledproducers["sensor"] = mockSensorProducer
	app.ledproducers[MULTI_BLOB_UID] = mockMultiBlobProducer
	app.sensorProd = []p.LedProducer{mockSensorProducer}

	ledReader := u.NewAtomicMapEvent[p.LedProducer]()
	app.stopsignal = make(chan struct{})
	ledBufferPool := &sync.Pool{
		New: func() any {
			return make([]p.Led, 10)
		},
	}

	app.shutdownWg.Add(1)
	go app.combineAndUpdateDisplay(ledReader, ledBufferPool)
	t.Cleanup(func() {
		close(app.stopsignal)
		app.shutdownWg.Wait()
	})

	// test initial state
	if len(mockPlatform.GetLastLeds()) != 0 {
		t.Errorf("Expected no leds to be written, but got %d", len(mockPlatform.GetLastLeds()))
	}

	// test sensor trigger
	mockPlatform.ClearLastLeds()
	mockSensorProducer.Start()
	ledReader.Send(mockSensorProducer.GetUID(), mockSensorProducer)
	time.Sleep(200 * time.Millisecond)
	if len(mockPlatform.GetLastLeds()) == 0 {
		t.Error("Expected leds to be written")
	}
}
