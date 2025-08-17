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
	lastLeds     [][]p.Led
	mu           sync.Mutex
	stopChan     chan struct{}
}

func (m *MockPlatform) SetLeds(leds []p.Led) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Make a copy of the slice to avoid data races
	ledsCopy := make([]p.Led, len(leds))
	copy(ledsCopy, leds)
	m.lastLeds = append(m.lastLeds, ledsCopy)
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
		sensorEvents: make(chan *u.Trigger),
		sensors:      make(map[string]c.SensorCfg),
		lastLeds:     make([][]p.Led, 0),
		stopChan:     make(chan struct{}),
	}
}

type MockLedProducer struct {
	*p.AbstractProducer
	uid          string
	wg           *sync.WaitGroup
	mu           sync.Mutex
	startCalls   int
	stopCalls    int
	triggerCalls int
	leds         []p.Led
}

func NewMockLedProducer(uid string, wg *sync.WaitGroup) *MockLedProducer {
	return &MockLedProducer{
		uid: uid,
		wg:  wg,
	}
}

func (m *MockLedProducer) Start() {
	m.mu.Lock()
	m.startCalls++
	m.mu.Unlock()
	if m.wg != nil {
		m.wg.Add(1) // Expect one sensor producer to run
	}
	// Simulate work and then signal completion
	go func() {
		time.Sleep(50 * time.Millisecond)
		if m.wg != nil {
			m.wg.Done()
		}
	}()
}

func (m *MockLedProducer) SendTrigger(trigger *u.Trigger) {
	m.mu.Lock()
	m.triggerCalls++
	m.wg.Add(1) // Expect one sensor producer to run
	m.mu.Unlock()
	// Simulate work and then signal completion
	go func() {
		time.Sleep(50 * time.Millisecond)
		m.wg.Done()
	}()
}

func (m *MockLedProducer) TryStop() (bool, error) {
	m.mu.Lock()
	m.stopCalls++
	m.mu.Unlock()
	return true, nil
}

func (m *MockLedProducer) GetLeds(buffer []p.Led) {
	copy(buffer, m.leds)
}

func (m *MockLedProducer) Exit() {}

func (m *MockLedProducer) GetUID() string {
	return m.uid
}

func (m *MockLedProducer) getCalls() (int, int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startCalls, m.stopCalls, m.triggerCalls
}

func TestStateManager(t *testing.T) {
	// Setup
	ossignal := make(chan os.Signal, 1)
	app := NewApp(ossignal)
	app.ledproducers = make(map[string]p.LedProducer)
	app.stopsignal = make(chan struct{})

	mockPlatform := NewMockPlatform()
	app.platform = mockPlatform
	mockPlatform.sensors["sensor1"] = c.SensorCfg{LedIndex: 0}

	permProd := NewMockLedProducer("perm", nil)
	sensorProd := NewMockLedProducer("sensor1", &app.sensorProdWg)
	afterProd := NewMockLedProducer("after", &app.afterProdWg)

	app.permProd = []p.LedProducer{permProd}
	app.sensorProd = []p.LedProducer{sensorProd}
	app.afterProd = []p.LedProducer{afterProd}
	app.ledproducers["perm"] = permProd
	app.ledproducers["sensor1"] = sensorProd
	app.ledproducers["after"] = afterProd

	// Mimic the behavior of initialise() where permanent producers are started first.
	for _, p := range app.permProd {
		p.Start()
	}

	app.shutdownWg.Add(1)
	go app.stateManager()
	t.Cleanup(func() {
		close(app.stopsignal)
		app.shutdownWg.Wait()
	})

	// --- Test Execution ---

	// 1. Initial state: perm producer should be running.
	start, stop, trigger := permProd.getCalls()
	if start != 1 || stop != 0 || trigger != 0 {
		t.Fatalf("Expected permProd to be running initially, got start:%d, stop:%d, trigger:%d", start, stop, trigger)
	}

	// 2. Trigger a sensor event
	mockPlatform.sensorEvents <- u.NewTrigger("sensor1", 100, time.Now())

	// 3. Verify state transition: perm should be stopped, sensor should be triggered
	time.Sleep(25 * time.Millisecond) // Allow time for state transition
	start, stop, trigger = permProd.getCalls()
	if start != 1 || stop != 1 || trigger != 0 {
		t.Fatalf("Expected permProd to be stopped, got start:%d, stop:%d, trigger:%d", start, stop, trigger)
	}
	start, stop, trigger = sensorProd.getCalls()
	if start != 0 || stop != 0 || trigger != 1 {
		t.Fatalf("Expected sensorProd to be triggered, got start:%d, stop:%d, trigger:%d", start, stop, trigger)
	}

	time.Sleep(75 * time.Millisecond) // Allow time for state transition

	// 4. Verify state transition: sensor done -> afterProd should start
	start, stop, trigger = afterProd.getCalls()
	if start != 1 || stop != 0 || trigger != 0 {
		t.Fatalf("Expected afterProd to be started, got start:%d, stop:%d, trigger:%d", start, stop, trigger)
	}

	time.Sleep(75 * time.Millisecond) // Allow time for state transition

	// 5. Verify state transition: afterProd done -> permProd should restart
	start, stop, trigger = permProd.getCalls()
	if start != 2 || stop != 1 || trigger != 0 {
		t.Fatalf("Expected permProd to be restarted, got start:%d, stop:%d, trigger:%d", start, stop, trigger)
	}
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

	mockSensorProducer := NewMockLedProducer("sensor", nil)
	mockMultiBlobProducer := NewMockLedProducer(MULTI_BLOB_UID, nil)
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
