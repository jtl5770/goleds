package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/platform"
	"lautenbacher.net/goleds/producer"
	"lautenbacher.net/goleds/util"
)

type MockPlatform struct {
	platform.Platform
	sensorEvents chan *util.Trigger
	sensors      map[string]config.SensorCfg
	lastLeds     [][]producer.Led
	mu           sync.Mutex
	stopChan     chan struct{}
}

func (m *MockPlatform) SetLeds(leds []producer.Led) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Make a copy of the slice to avoid data races
	ledsCopy := make([]producer.Led, len(leds))
	copy(ledsCopy, leds)
	m.lastLeds = append(m.lastLeds, ledsCopy)
}

func (m *MockPlatform) GetSensorEvents() <-chan *util.Trigger {
	return m.sensorEvents
}

func (m *MockPlatform) GetSensorLedIndices() map[string]int {
	indices := make(map[string]int)
	for uid, cfg := range m.sensors {
		indices[uid] = cfg.LedIndex
	}
	return indices
}

func (m *MockPlatform) Start(pool *sync.Pool, calibCurves platform.CalibrationCurves) error {
	// The new platform interface doesn't require a goroutine here for the mock.
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

func (m *MockPlatform) Calibrate(calibCurves platform.CalibrationCurves) error {
	return nil
}

func (m *MockPlatform) IsCalibrating() bool {
	return false
}

func (m *MockPlatform) GetLastLeds() [][]producer.Led {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to avoid race conditions
	ret := make([][]producer.Led, len(m.lastLeds))
	for i, leds := range m.lastLeds {
		ret[i] = make([]producer.Led, len(leds))
		copy(ret[i], leds)
	}
	return ret
}

func NewMockPlatform() *MockPlatform {
	return &MockPlatform{
		sensorEvents: make(chan *util.Trigger, 10),
		sensors:      make(map[string]config.SensorCfg),
		stopChan:     make(chan struct{}),
	}
}

type MockLedProducer struct {
	uid          string
	leds         []producer.Led
	priority     int
	active       bool
	events       *util.AtomicMapEvent[producer.LedProducer]
	triggerEvent *util.AtomicEvent[*util.Trigger]
	mu           sync.Mutex
	stopChan     chan bool
}

func NewMockLedProducer(uid string, events *util.AtomicMapEvent[producer.LedProducer]) *MockLedProducer {
	return &MockLedProducer{
		uid:          uid,
		leds:         make([]producer.Led, 10),
		active:       true,
		events:       events,
		triggerEvent: util.NewAtomicEvent[*util.Trigger](),
		stopChan:     make(chan bool, 1),
	}
}

func (m *MockLedProducer) ClearLeds() {
	m.mu.Lock()
	defer m.mu.Unlock()
	clear(m.leds)
	if m.events != nil {
		m.events.Send(m.uid, m)
	}
}

func (m *MockLedProducer) GetLeds(buffer []producer.Led) {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy(buffer, m.leds)
}

func (m *MockLedProducer) GetUID() string {
	return m.uid
}

func (m *MockLedProducer) GetPriority() int {
	return m.priority
}

func (m *MockLedProducer) SetPriority(priority int) {
	m.priority = priority
}

func (m *MockLedProducer) IsActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active
}

func (m *MockLedProducer) SetActive(active bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active = active
}

func (m *MockLedProducer) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active = true
}

func (m *MockLedProducer) Exit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active = false
}

func (m *MockLedProducer) SendTrigger(trigger *util.Trigger) {
	m.triggerEvent.Send(trigger)
}

func (m *MockLedProducer) TryStop() (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active = false
	return true, nil
}

func (m *MockLedProducer) SetLeds(leds []producer.Led) {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy(m.leds, leds)
	if m.events != nil {
		m.events.Send(m.uid, m)
	}
}

func TestStateManager_SensorEvent_TransitionsToSensorState(t *testing.T) {
	ossignal := make(chan os.Signal, 1)
	app := NewApp(ossignal)
	app.stopsignal = make(chan struct{})
	app.ledproducers = make(map[string]producer.LedProducer)

	mockPlatform := NewMockPlatform()
	app.platform = mockPlatform
	t.Cleanup(mockPlatform.Stop)

	mockSensorProducer := NewMockLedProducer("sensor", nil)
	app.ledproducers["sensor"] = mockSensorProducer
	app.sensorProd = []producer.LedProducer{mockSensorProducer}

	mockAfterProducer := NewMockLedProducer(MultiBlobUID, nil)
	app.ledproducers[MultiBlobUID] = mockAfterProducer
	app.afterProd = []producer.LedProducer{mockAfterProducer}

	app.shutdownWg.Add(1)
	go app.stateManager()
	t.Cleanup(func() {
		close(app.stopsignal)
		app.shutdownWg.Wait()
	})

	trigger := util.NewTrigger("sensor", 100, time.Now())
	mockPlatform.sensorEvents <- trigger

	// Verify that the trigger was received by the producer
	select {
	case <-mockSensorProducer.triggerEvent.Channel():
		receivedTrigger := mockSensorProducer.triggerEvent.Value()
		if receivedTrigger != trigger {
			t.Errorf("Expected trigger %v, got %v", trigger, receivedTrigger)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for trigger event")
	}
}

func TestStateManager_SensorProducerCompletion_TransitionsToAfterProdState(t *testing.T) {
	ossignal := make(chan os.Signal, 1)
	app := NewApp(ossignal)
	app.stopsignal = make(chan struct{})
	app.ledproducers = make(map[string]producer.LedProducer)

	mockPlatform := NewMockPlatform()
	app.platform = mockPlatform
	t.Cleanup(mockPlatform.Stop)

	mockSensorProducer := NewMockLedProducer("sensor", nil)
	app.ledproducers["sensor"] = mockSensorProducer
	app.sensorProd = []producer.LedProducer{mockSensorProducer}

	mockAfterProducer := NewMockLedProducer(MultiBlobUID, nil)
	mockAfterProducer.SetActive(false)
	app.ledproducers[MultiBlobUID] = mockAfterProducer
	app.afterProd = []producer.LedProducer{mockAfterProducer}

	app.shutdownWg.Add(1)
	go app.stateManager()
	t.Cleanup(func() {
		close(app.stopsignal)
		app.shutdownWg.Wait()
	})

	// Simulate a sensor run by adding to the waitgroup, triggering, and then marking as done
	app.sensorProdWg.Add(1)
	trigger := util.NewTrigger("sensor", 100, time.Now())
	mockPlatform.sensorEvents <- trigger
	app.sensorProdWg.Done()

	// Wait for the state transition to afterProd and verify that the after-producer is started
	var afterProdStarted bool
	for range 50 {
		if mockAfterProducer.IsActive() {
			afterProdStarted = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !afterProdStarted {
		t.Error("Expected after-producer to be started after sensor producer completion")
	}
}

func TestStateManager_PermanentProducersRestart_AfterAfterProdCompletion(t *testing.T) {
	ossignal := make(chan os.Signal, 1)
	app := NewApp(ossignal)
	app.stopsignal = make(chan struct{})
	app.ledproducers = make(map[string]producer.LedProducer)

	mockPlatform := NewMockPlatform()
	app.platform = mockPlatform
	t.Cleanup(mockPlatform.Stop)

	mockSensorProducer := NewMockLedProducer("sensor", nil)
	app.ledproducers["sensor"] = mockSensorProducer
	app.sensorProd = []producer.LedProducer{mockSensorProducer}

	mockAfterProducer := NewMockLedProducer(MultiBlobUID, nil)
	app.ledproducers[MultiBlobUID] = mockAfterProducer
	app.afterProd = []producer.LedProducer{mockAfterProducer}

	mockPermProducer := NewMockLedProducer(NightLedUID, nil)
	app.ledproducers[NightLedUID] = mockPermProducer
	app.permProd = []producer.LedProducer{mockPermProducer}

	app.shutdownWg.Add(1)
	go app.stateManager()
	t.Cleanup(func() {
		close(app.stopsignal)
		app.shutdownWg.Wait()
	})

	// Start a permanent producer
	for _, p := range app.permProd {
		p.Start()
	}

	// 1. Trigger a sensor event (transitions to stateSensor)
	app.sensorProdWg.Add(1)
	trigger := util.NewTrigger("sensor", 100, time.Now())
	mockPlatform.sensorEvents <- trigger

	// Verify permanent producer was stopped
	var permStopped bool
	for range 50 {
		if !mockPermProducer.IsActive() {
			permStopped = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !permStopped {
		t.Error("Expected permanent producer to be stopped on sensor trigger")
	}

	// 2. Complete the sensor run (transitions to stateAfterProd)
	app.afterProdWg.Add(1) // Add to afterProdWg BEFORE completing sensor run
	app.sensorProdWg.Done()

	// Wait for after-producer to start
	var afterProdStarted bool
	for range 50 {
		if mockAfterProducer.IsActive() {
			afterProdStarted = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !afterProdStarted {
		t.Fatal("Expected after-producer to be started")
	}

	// 3. Complete the after-producer run (transitions to stateIdle)
	app.afterProdWg.Done()

	// Verify permanent producer was restarted
	var permProdRestarted bool
	for range 50 {
		if mockPermProducer.IsActive() {
			permProdRestarted = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !permProdRestarted {
		t.Error("Expected permanent producer to be restarted after after-producer completion")
	}
}

func TestCombineAndUpdateDisplay(t *testing.T) {
	ossignal := make(chan os.Signal, 1)
	app := NewApp(ossignal)
	app.stopsignal = make(chan struct{})
	app.ledproducers = make(map[string]producer.LedProducer)

	mockPlatform := NewMockPlatform()
	app.platform = mockPlatform
	// Start the mock platform to begin capturing LED data
	mockPlatform.Start(nil, nil)
	t.Cleanup(mockPlatform.Stop)

	mockPlatform.sensors["sensor"] = config.SensorCfg{LedIndex: 0, SpiMultiplex: "", AdcChannel: 0}

	mockSensorProducer := NewMockLedProducer("sensor", nil)
	mockMultiBlobProducer := NewMockLedProducer(MultiBlobUID, nil)
	app.ledproducers["sensor"] = mockSensorProducer
	app.ledproducers[MultiBlobUID] = mockMultiBlobProducer
	app.sensorProd = []producer.LedProducer{mockSensorProducer}
	app.afterProd = []producer.LedProducer{mockMultiBlobProducer}

	ledBufferPool := &sync.Pool{
		New: func() any {
			return make([]producer.Led, 10)
		},
	}

	ledReader := util.NewAtomicMapEvent[producer.LedProducer]()

	app.shutdownWg.Add(1)
	go app.combineAndUpdateDisplay(ledReader, ledBufferPool)
	t.Cleanup(func() {
		close(app.stopsignal)
		app.shutdownWg.Wait()
	})

	// Set initial LED values on the producer
	initialLeds := make([]producer.Led, 10)
	initialLeds[0] = producer.Led{Red: 255, Green: 0, Blue: 0}
	mockSensorProducer.SetLeds(initialLeds)

	// Send an event to trigger combineAndUpdateDisplay
	ledReader.Send("sensor", mockSensorProducer)

	// Allow some time for the goroutine to process the event
	time.Sleep(50 * time.Millisecond)

	// Verify that the platform received the correct LED data
	lastLeds := mockPlatform.GetLastLeds()
	if len(lastLeds) == 0 {
		t.Fatal("Platform did not receive any LED data")
	}

	receivedLeds := lastLeds[len(lastLeds)-1]
	if receivedLeds[0].Red != 255 || receivedLeds[0].Green != 0 || receivedLeds[0].Blue != 0 {
		t.Errorf("Expected LED 0 to be red, got %v", receivedLeds[0])
	}
}

func TestWatchConfigFile_Detection(t *testing.T) {
	tempDir := t.TempDir()
	cfgFile := filepath.Join(tempDir, "test_watch_config.yml")

	if err := os.WriteFile(cfgFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	reloadEvent := util.NewAtomicEvent[bool]()
	go watchConfigFile(cfgFile, reloadEvent)

	// Wait for watcher to register
	time.Sleep(50 * time.Millisecond)

	// Modify file
	if err := os.WriteFile(cfgFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	select {
	case <-reloadEvent.Channel():
		// Succeeded
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for file reload event")
	}
}

