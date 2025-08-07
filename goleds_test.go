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
	sensorEvents chan *u.Trigger
	sensors      map[string]c.SensorCfg
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

func (m *MockPlatform) Start(ledWriter chan []p.Led, pool *sync.Pool) error {
	return nil
}

func (m *MockPlatform) Stop() {
}

func (m *MockPlatform) DisplayLeds(leds []p.Led) {
	// do nothing in mock
}

func (m *MockPlatform) GetForceUpdateDelay() time.Duration {
	return 1 * time.Second
}

func (m *MockPlatform) GetLedsTotal() int {
	return 10
}

func NewMockPlatform() *MockPlatform {
	return &MockPlatform{
		sensorEvents: make(chan *u.Trigger),
		sensors:      make(map[string]c.SensorCfg),
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

func TestFireController(t *testing.T) {
	ossignal := make(chan os.Signal, 1)
	app := NewApp(ossignal)
	app.ledproducers = make(map[string]p.LedProducer)

	mockPlatform := NewMockPlatform()
	app.platform = mockPlatform

	mockProducer := NewMockLedProducer("test")
	app.ledproducers["test"] = mockProducer

	app.stopsignal = make(chan struct{})
	app.shutdownWg.Add(1)
	// go app.fireController()
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

	mockPlatform.sensors["sensor"] = c.SensorCfg{LedIndex: 0, SpiMultiplex: "", AdcChannel: 0, TriggerValue: 0}

	mockSensorProducer := NewMockLedProducer("sensor")
	mockMultiBlobProducer := NewMockLedProducer(MULTI_BLOB_UID)
	app.ledproducers["sensor"] = mockSensorProducer
	app.ledproducers[MULTI_BLOB_UID] = mockMultiBlobProducer
	app.sensorProd = []p.LedProducer{mockSensorProducer}

	ledReader := u.NewAtomicMapEvent[p.LedProducer]()
	ledWriter := make(chan []p.Led, 1)
	app.stopsignal = make(chan struct{})
	ledBufferPool := &sync.Pool{
		New: func() any {
			return make([]p.Led, 10)
		},
	}

	app.shutdownWg.Add(1)
	go app.combineAndUpdateDisplay(ledReader, ledWriter, ledBufferPool)
	t.Cleanup(func() {
		close(app.stopsignal)
		app.shutdownWg.Wait()
	})

	// test initial state
	select {
	case <-ledWriter:
		t.Error("Expected no leds to be written")
	default:
	}

	// test sensor trigger
	mockSensorProducer.Start()
	ledReader.Send(mockSensorProducer.GetUID(), mockSensorProducer)
	time.Sleep(100 * time.Millisecond)
	select {
	case <-ledWriter:
		// expected
	default:
		t.Error("Expected leds to be written")
	}

	// test stop
	mockSensorProducer.Start()
	ledReader.Send(mockSensorProducer.GetUID(), mockSensorProducer)
	time.Sleep(100 * time.Millisecond)
}
