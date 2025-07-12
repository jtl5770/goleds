package main

import (
	"sync"
	"testing"
	"time"

	c "lautenbacher.net/goleds/config"
	d "lautenbacher.net/goleds/driver"
	p "lautenbacher.net/goleds/producer"
	u "lautenbacher.net/goleds/util"
)

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
	oldConfig := c.CONFIG
	c.CONFIG.HoldLED.Enabled = true
	c.CONFIG.HoldLED.TriggerValue = 100
	c.CONFIG.HoldLED.TriggerDelay = 1 * time.Second
	t.Cleanup(func() {
		c.CONFIG = oldConfig
	})

	oldSensorReader := d.SensorReader
	d.SensorReader = make(chan *d.Trigger)
	t.Cleanup(func() {
		d.SensorReader = oldSensorReader
	})

	oldLedProducers := ledproducers
	ledproducers = make(map[string]p.LedProducer)
	mockProducer := NewMockLedProducer("test")
	ledproducers["test"] = mockProducer
	mockHoldProducer := NewMockLedProducer(HOLD_LED_UID)
	ledproducers[HOLD_LED_UID] = mockHoldProducer
	t.Cleanup(func() {
		ledproducers = oldLedProducers
	})

	stopsignal = make(chan bool)
	var wg sync.WaitGroup
	wg.Add(1)
	go fireController(stopsignal, &wg)
	t.Cleanup(func() {
		close(stopsignal)
		wg.Wait()
	})

	// test normal trigger
	d.SensorReader <- d.NewTrigger("test", 10, time.Now())
	time.Sleep(100 * time.Millisecond)
	if !mockProducer.GetIsRunning() {
		t.Error("Expected producer to be running")
	}
	mockProducer.Stop()

	// test hold trigger
	now := time.Now()
	// first trigger, should not start hold producer
	d.SensorReader <- d.NewTrigger("holdtest", 110, now)
	time.Sleep(100 * time.Millisecond)
	if mockHoldProducer.GetIsRunning() {
		t.Fatal("Expected hold producer to not be running yet")
	}

	// second trigger in the time window, should start hold producer
	d.SensorReader <- d.NewTrigger("holdtest", 110, now.Add(c.CONFIG.HoldLED.TriggerDelay+200*time.Millisecond))
	time.Sleep(100 * time.Millisecond)
	if !mockHoldProducer.GetIsRunning() {
		t.Fatal("Expected hold producer to be running")
	}

	// third trigger in the time window, should stop hold producer
	d.SensorReader <- d.NewTrigger("holdtest", 110, now.Add(2*(c.CONFIG.HoldLED.TriggerDelay+200*time.Millisecond)))
	time.Sleep(100 * time.Millisecond)
	if mockHoldProducer.GetIsRunning() {
		t.Fatal("Expected hold producer to be stopped")
	}
}

func TestCombineAndUpdateDisplay(t *testing.T) {
	// setup
	oldConfig := c.CONFIG
	c.CONFIG = c.Config{} // Reset config
	c.CONFIG.Hardware.Display.LedsTotal = 10
	c.CONFIG.Hardware.Display.ForceUpdateDelay = 1 * time.Second
	c.CONFIG.MultiBlobLED.Enabled = true
	t.Cleanup(func() {
		c.CONFIG = oldConfig
	})

	oldSensors := d.Sensors
	d.Sensors = map[string]*d.Sensor{"sensor": d.NewSensor("sensor", 0, "", 0, 0)}
	t.Cleanup(func() {
		d.Sensors = oldSensors
	})

	oldLedProducers := ledproducers
	ledproducers = make(map[string]p.LedProducer)
	mockSensorProducer := NewMockLedProducer("sensor")
	mockMultiBlobProducer := NewMockLedProducer(MULTI_BLOB_UID)
	ledproducers["sensor"] = mockSensorProducer
	ledproducers[MULTI_BLOB_UID] = mockMultiBlobProducer
	t.Cleanup(func() {
		ledproducers = oldLedProducers
	})

	ledReader := u.NewAtomicEvent[p.LedProducer]()
	ledWriter := make(chan []p.Led, 1)
	stopsignal = make(chan bool)

	var wg sync.WaitGroup
	wg.Add(1)
	go combineAndUpdateDisplay(ledReader, ledWriter, stopsignal, &wg)
	t.Cleanup(func() {
		close(stopsignal)
		wg.Wait()
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
