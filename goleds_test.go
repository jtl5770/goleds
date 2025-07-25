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

func (m *MockPlatform) Start() error {
	return nil
}

func (m *MockPlatform) Stop() {
}

func (m *MockPlatform) DisplayDriver(display chan []p.Led, stopSignal chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-stopSignal:
			return
		case <-display:
		}
	}
}

func (m *MockPlatform) SensorDriver(stopSignal chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	<-stopSignal
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
	// do nothing
}

func (m *MockLedProducer) Stop() {
	m.isRunning = false
}

func (m *MockLedProducer) GetIsRunning() bool {
	return m.isRunning
}

func (m *MockLedProducer) GetLeds() []p.Led {
	return m.leds
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

	app.stopsignal = make(chan bool)
	app.shutdownWg.Add(1)
	go app.fireController()
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
	mockProducer.Stop()
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

	ledReader := u.NewAtomicEvent[p.LedProducer]()
	ledWriter := make(chan []p.Led, 1)
	app.stopsignal = make(chan bool)

	app.shutdownWg.Add(1)
	go app.combineAndUpdateDisplay(ledReader, ledWriter)
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
	ledReader.Send(mockSensorProducer)
	time.Sleep(100 * time.Millisecond)
	select {
	case <-ledWriter:
		// expected
	default:
		t.Error("Expected leds to be written")
	}

	// test stop
	mockSensorProducer.Start()
	ledReader.Send(mockSensorProducer)
	time.Sleep(100 * time.Millisecond)
	if mockMultiBlobProducer.GetIsRunning() {
		t.Error("Expected multiblob producer to be stopped")
	}
}
