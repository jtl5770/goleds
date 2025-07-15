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
	sensorEvents chan *pl.Trigger
	sensors      map[string]c.SensorCfg
}

func (m *MockPlatform) GetSensorEvents() <-chan *pl.Trigger {
	return m.sensorEvents
}

func (m *MockPlatform) GetSensors() map[string]c.SensorCfg {
	return m.sensors
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

func NewMockPlatform() *MockPlatform {
	return &MockPlatform{
		sensorEvents: make(chan *pl.Trigger),
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
	// setup
	cfile := ""
	realp := false
	sensp := false
	ossignal := make(chan os.Signal, 1)
	app := NewApp(&cfile, &realp, &sensp, ossignal)
	app.ledproducers = make(map[string]p.LedProducer)

	mockPlatform := NewMockPlatform()
	app.platform = mockPlatform

	triggerValue := 100
	triggerDelay := 1 * time.Second

	mockProducer := NewMockLedProducer("test")
	app.ledproducers["test"] = mockProducer
	mockHoldProducer := NewMockLedProducer(HOLD_LED_UID)
	app.ledproducers[HOLD_LED_UID] = mockHoldProducer

	app.stopsignal = make(chan bool)
	app.shutdownWg.Add(1)
	go app.fireController(true, triggerDelay, triggerValue)
	t.Cleanup(func() {
		close(app.stopsignal)
		app.shutdownWg.Wait()
	})

	// test normal trigger
	mockPlatform.sensorEvents <- pl.NewTrigger("test", 10, time.Now())
	time.Sleep(100 * time.Millisecond)
	if !mockProducer.GetIsRunning() {
		t.Error("Expected producer to be running")
	}
	mockProducer.Stop()

	// test hold trigger
	now := time.Now()
	// first trigger, should not start hold producer
	mockPlatform.sensorEvents <- pl.NewTrigger(HOLD_LED_UID, 110, now)
	time.Sleep(100 * time.Millisecond)
	if mockHoldProducer.GetIsRunning() {
		t.Fatal("Expected hold producer to not be running yet")
	}

	// second trigger in the time window, should start hold producer
	mockPlatform.sensorEvents <- pl.NewTrigger(HOLD_LED_UID, 110, now.Add(triggerDelay+200*time.Millisecond))
	time.Sleep(100 * time.Millisecond)
	if !mockHoldProducer.GetIsRunning() {
		t.Fatal("Expected hold producer to be running")
	}

	// third trigger in the time window, should stop hold producer
	mockPlatform.sensorEvents <- pl.NewTrigger(HOLD_LED_UID, 110, now.Add(2*(triggerDelay+200*time.Millisecond)))
	time.Sleep(100 * time.Millisecond)
	if mockHoldProducer.GetIsRunning() {
		t.Fatal("Expected hold producer to be stopped")
	}
}

func TestCombineAndUpdateDisplay(t *testing.T) {
	// setup
	cfile := ""
	realp := false
	sensp := false
	ossignal := make(chan os.Signal, 1)
	app := NewApp(&cfile, &realp, &sensp, ossignal)
	app.ledproducers = make(map[string]p.LedProducer)

	ledsTotal := 10
	forceUpdateDelay := 1 * time.Second

	mockPlatform := NewMockPlatform()
	app.platform = mockPlatform

	mockPlatform.sensors["sensor"] = c.SensorCfg{LedIndex: 0, SpiMultiplex: "", AdcChannel: 0, TriggerValue: 0}

	mockSensorProducer := NewMockLedProducer("sensor")
	mockMultiBlobProducer := NewMockLedProducer(MULTI_BLOB_UID)
	app.ledproducers["sensor"] = mockSensorProducer
	app.ledproducers[MULTI_BLOB_UID] = mockMultiBlobProducer

	ledReader := u.NewAtomicEvent[p.LedProducer]()
	ledWriter := make(chan []p.Led, 1)
	app.stopsignal = make(chan bool)

	app.shutdownWg.Add(1)
	go app.combineAndUpdateDisplay(true, false, false, true, false, ledReader, ledWriter, ledsTotal, forceUpdateDelay)
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

	// test multiblob trigger
	mockSensorProducer.Stop()
	ledReader.Send(mockSensorProducer)
	time.Sleep(100 * time.Millisecond)
	if !mockMultiBlobProducer.GetIsRunning() {
		t.Error("Expected multiblob producer to be running")
	}

	// test stop
	mockSensorProducer.Start()
	ledReader.Send(mockSensorProducer)
	time.Sleep(100 * time.Millisecond)
	if mockMultiBlobProducer.GetIsRunning() {
		t.Error("Expected multiblob producer to be stopped")
	}
}
