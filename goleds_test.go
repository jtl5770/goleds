package main

import (
	"os"
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

func (m *MockPlatform) Calibrate() error {
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

func (m *MockPlatform) ClearLastLeds() {
	m.mu.Lock()
	m.lastLeds = nil
	m.mu.Unlock()
}

func NewMockPlatform() *MockPlatform {
	return &MockPlatform{
		sensorEvents: make(chan *util.Trigger),
		sensors:      make(map[string]config.SensorCfg),
		lastLeds:     make([][]producer.Led, 0),
		stopChan:     make(chan struct{}),
	}
}

type MockLedProducer struct {
	*producer.AbstractProducer
	uid          string
	wg           *sync.WaitGroup
	mu           sync.Mutex
	startCalls   int
	stopCalls    int
	triggerCalls int
	leds         []producer.Led
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

func (m *MockLedProducer) SendTrigger(trigger *util.Trigger) {
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

func (m *MockLedProducer) GetLeds(buffer []producer.Led) {
	copy(buffer, m.leds)
}

func (m *MockLedProducer) Exit() {}

func (m *MockLedProducer) GetUID() string {
	return m.uid
}

func (m *MockLedProducer) GetPriority() int {
	return 0
}

func (m *MockLedProducer) IsActive() bool {
	return true
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
	app.ledproducers = make(map[string]producer.LedProducer)
	app.stopsignal = make(chan struct{})

	mockPlatform := NewMockPlatform()
	app.platform = mockPlatform
	mockPlatform.sensors["sensor1"] = config.SensorCfg{LedIndex: 0}

	permProd := NewMockLedProducer("perm", nil)
	sensorProd := NewMockLedProducer("sensor1", &app.sensorProdWg)
	afterProd := NewMockLedProducer("after", &app.afterProdWg)

	app.permProd = []producer.LedProducer{permProd}
	app.sensorProd = []producer.LedProducer{sensorProd}
	app.afterProd = []producer.LedProducer{afterProd}
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
	mockPlatform.sensorEvents <- util.NewTrigger("sensor1", 100, time.Now())

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
	app.ledproducers = make(map[string]producer.LedProducer)

	mockPlatform := NewMockPlatform()
	app.platform = mockPlatform
	// Start the mock platform to begin capturing LED data
	mockPlatform.Start(nil)
	t.Cleanup(mockPlatform.Stop)

	mockPlatform.sensors["sensor"] = config.SensorCfg{LedIndex: 0, SpiMultiplex: "", AdcChannel: 0}

	mockSensorProducer := NewMockLedProducer("sensor", nil)
	mockMultiBlobProducer := NewMockLedProducer(MultiBlobUID, nil)
	app.ledproducers["sensor"] = mockSensorProducer
	app.ledproducers[MultiBlobUID] = mockMultiBlobProducer
	app.sensorProd = []producer.LedProducer{mockSensorProducer}

	ledReader := util.NewAtomicMapEvent[producer.LedProducer]()
	app.stopsignal = make(chan struct{})
	ledBufferPool := &sync.Pool{
		New: func() any {
			return make([]producer.Led, 10)
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

func TestHashLEDs(t *testing.T) {
	leds1 := []producer.Led{
		{Red: 100, Green: 150, Blue: 200},
		{Red: 0, Green: 50, Blue: 255},
	}
	leds2 := []producer.Led{
		{Red: 100, Green: 150, Blue: 200},
		{Red: 0, Green: 50, Blue: 255},
	}
	leds3 := []producer.Led{
		{Red: 100, Green: 150, Blue: 200},
		{Red: 0, Green: 51, Blue: 255},
	}

	h1 := hashLEDs(leds1)
	h2 := hashLEDs(leds2)
	h3 := hashLEDs(leds3)

	if h1 != h2 {
		t.Errorf("Expected identical LED slices to have matching hashes, got %d vs %d", h1, h2)
	}
	if h1 == h3 {
		t.Errorf("Expected different LED slices to have distinct hashes, got identical hash %d", h1)
	}
}

func BenchmarkHashLEDs(b *testing.B) {
	leds := make([]producer.Led, 100)
	for i := range leds {
		leds[i] = producer.Led{Red: float64(i % 256), Green: float64((i * 2) % 256), Blue: float64((i * 3) % 256)}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = hashLEDs(leds)
	}
}
