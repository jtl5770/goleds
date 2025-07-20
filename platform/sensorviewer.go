package platform

import (
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/gammazero/deque"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lautenbacher.net/goleds/config"
)

const (
	maxSensorHistory = 500
	viewerTitle      = " GOLEDS Sensor Viewer "
)

// SensorViewer is a TUI component for displaying real-time sensor data.
type SensorViewer struct {
	app          *tview.Application
	view         *tview.TextView
	sensorValues map[string]*deque.Deque[int]
	sensorNames  []string
	mu           sync.Mutex
	ossignal     chan os.Signal
	introText    string
}

type sensorStats struct {
	min    int
	max    int
	mean   float64
	median float64
	stdDev float64
}

// NewSensorViewer creates and initializes a new SensorViewer.
func NewSensorViewer(sensorCfgs map[string]config.SensorCfg, ossignal chan os.Signal, introText string) *SensorViewer {
	sv := &SensorViewer{
		app:          tview.NewApplication(),
		sensorValues: make(map[string]*deque.Deque[int]),
		sensorNames:  make([]string, 0, len(sensorCfgs)),
		ossignal:     ossignal,
		introText:    introText,
	}

	for name := range sensorCfgs {
		sv.sensorNames = append(sv.sensorNames, name)
		sv.sensorValues[name] = new(deque.Deque[int])
		sv.sensorValues[name].Grow(maxSensorHistory)
	}
	// Sort names for consistent display order
	sort.Strings(sv.sensorNames)

	return sv
}

// Start initializes and runs the TUI. It should be called as a goroutine.
func (sv *SensorViewer) Start(stopSignal chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	sv.setupUI()

	// Goroutine to handle shutdown
	go func() {
		<-stopSignal
		log.Println("Stopping SensorViewer TUI...")
		sv.app.Stop()
	}()

	if err := sv.app.Run(); err != nil {
		log.Fatalf("Error running SensorViewer TUI: %v", err)
	}
	log.Println("SensorViewer TUI has stopped.")
}

// Update receives the latest sensor values, updates the internal state,
// and redraws the TUI. This method is safe for concurrent use.
func (sv *SensorViewer) Update(latestValues map[string]int) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	for name, value := range latestValues {
		if q, ok := sv.sensorValues[name]; ok {
			if q.Len() == maxSensorHistory {
				q.PopFront()
			}
			q.PushBack(value)
		}
	}

	// Redraw the view in the main TUI thread
	sv.app.QueueUpdateDraw(func() {
		sv.draw()
	})
}

func (sv *SensorViewer) setupUI() {
	sv.view = tview.NewTextView()
	sv.view.SetDynamicColors(true)
	sv.view.SetTextAlign(tview.AlignLeft)
	sv.view.SetBackgroundColor(tcell.ColorDarkSlateGray)
	sv.view.SetBorder(true).SetTitle(viewerTitle).SetTitleColor(tcell.ColorLightBlue)

	intro := tview.NewTextView()
	intro.SetBorder(true).SetTitle(" GOLEDS Simulation ").SetTitleColor(tcell.ColorLightBlue)
	intro.SetText(sv.introText)
	intro.SetTextAlign(1)
	intro.SetDynamicColors(true)
	intro.SetBackgroundColor(tcell.ColorDarkSlateGray)

	numSensors := len(sv.sensorNames)
	layout := tview.NewFlex().SetDirection(tview.FlexRow)
	layout.AddItem(intro, 4, 1, false)
	layout.AddItem(sv.view, numSensors*2+3, 1, true)
	layout.SetRect(1, 1, int(math.Max(float64(numSensors*15+24), 70)), numSensors*2+10)

	sv.app.SetRoot(layout, true).SetFocus(sv.view)
	sv.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		key := string(event.Rune())
		if key == "q" || key == "Q" {
			sv.app.Stop()
			sv.ossignal <- os.Interrupt
		} else if key == "r" || key == "R" {
			sv.app.Stop()
			sv.ossignal <- syscall.SIGHUP
		}
		return event
	})
}

// draw updates the TextView with the current sensor data.
// This must be called from within the TUI's main thread.
func (sv *SensorViewer) draw() {
	var b strings.Builder
	b.WriteString("[yellow]Sensor            Last    Min    Max     Mean   Median   StdDev[white]\n")
	for _, name := range sv.sensorNames {
		if values, ok := sv.sensorValues[name]; ok && values.Len() > 0 {
			lastVal := values.Back()

			// Create a slice copy for calculations
			data := make([]int, values.Len())
			for i := 0; i < values.Len(); i++ {
				data[i] = values.At(i)
			}

			stats := calculateStats(data)

			b.WriteString(fmt.Sprintf("%-18s %5d %5d %5d %8.2f %8.2f %8.2f\n",
				name, lastVal, stats.min, stats.max, stats.mean, stats.median, stats.stdDev))
		} else {
			b.WriteString(fmt.Sprintf("%-18s (no data)\n", name))
		}
	}
	sv.view.SetText(b.String())
}

func calculateStats(data []int) sensorStats {
	if len(data) == 0 {
		return sensorStats{}
	}

	// Min, Max, Sum
	var sum int
	min, max := data[0], data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}

	// Mean
	mean := float64(sum) / float64(len(data))

	// Median
	sort.Ints(data)
	var median float64
	mid := len(data) / 2
	if len(data)%2 == 0 {
		median = float64(data[mid-1]+data[mid]) / 2.0
	} else {
		median = float64(data[mid])
	}

	// Standard Deviation
	var sumOfSquares float64
	for _, v := range data {
		sumOfSquares += (float64(v) - mean) * (float64(v) - mean)
	}
	stdDev := math.Sqrt(sumOfSquares / float64(len(data)))

	return sensorStats{
		min:    min,
		max:    max,
		mean:   mean,
		median: median,
		stdDev: stdDev,
	}
}
