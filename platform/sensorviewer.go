package platform

import (
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gammazero/deque"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	c "lautenbacher.net/goleds/config"
)

const (
	maxSensorHistory = 500
	viewerTitle      = " GOLEDS Sensor Viewer "
	colWidth         = 18 // Width for each sensor's data column
)

// SensorViewer is a TUI component for displaying real-time sensor data.
type SensorViewer struct {
	tuiApp       *tview.Application
	view         *tview.TextView
	sensorValues map[string]*deque.Deque[int]
	sensorCfgs   map[string]c.SensorCfg
	sensorNames  []string
	mu           sync.Mutex
	ossignal     chan os.Signal
	devMode      bool
}

type sensorStats struct {
	min    int
	max    int
	mean   float64
	median float64
	stdDev float64
}

// NewSensorViewer creates and initializes a new SensorViewer.
func NewSensorViewer(sensorCfgs map[string]c.SensorCfg, ossignal chan os.Signal, devMode bool) *SensorViewer {
	sv := &SensorViewer{
		tuiApp:       tview.NewApplication(),
		sensorValues: make(map[string]*deque.Deque[int]),
		sensorCfgs:   sensorCfgs,
		sensorNames:  make([]string, 0, len(sensorCfgs)),
		ossignal:     ossignal,
		devMode:      devMode,
	}

	for name := range sensorCfgs {
		sv.sensorNames = append(sv.sensorNames, name)
		sv.sensorValues[name] = new(deque.Deque[int])
		sv.sensorValues[name].Grow(maxSensorHistory)
	}
	// Sort sensorNames based on the LedIndex from sensorCfgs to match the old layout.
	sort.Slice(sv.sensorNames, func(i, j int) bool {
		nameI := sv.sensorNames[i]
		nameJ := sv.sensorNames[j]
		return sv.sensorCfgs[nameI].LedIndex < sv.sensorCfgs[nameJ].LedIndex
	})

	return sv
}

// Start initializes and runs the TUI. It should be called as a goroutine.
func (sv *SensorViewer) Start(stopSignal chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	sv.setupUI()

	// Goroutine to handle shutdown
	go func() {
		<-stopSignal
		slog.Info("Stopping SensorViewer TUI...")
		sv.tuiApp.Stop()
	}()

	if err := sv.tuiApp.Run(); err != nil {
		slog.Error("Error running SensorViewer TUI", "error", err)
		os.Exit(1)
	}
	slog.Info("SensorViewer TUI has stopped.")
}

// Update receives the latest sensor values, prepares the display strings,
// and schedules a TUI redraw. This method is safe for concurrent use.
func (sv *SensorViewer) Update(latestValues map[string]int) {
	sv.mu.Lock()

	for name, value := range latestValues {
		if q, ok := sv.sensorValues[name]; ok {
			if q.Len() == maxSensorHistory {
				q.PopFront()
			}
			q.PushBack(value)
		}
	}

	// Prepare display strings while still under the lock
	line1, line2, line3 := sv.prepareDisplayStrings()

	sv.mu.Unlock()

	// Redraw the view in the main TUI thread, passing the prepared data via a closure.
	sv.tuiApp.QueueUpdateDraw(func() {
		sv.draw(line1, line2, line3)
	})
}

// runSensorDataGenerator is used only during development of this
// component to feed random data to the SensorViewer without the need
// for real hardware.
func (sv *SensorViewer) RunSensorDataGenForDev(loopDelay time.Duration, stopSignal chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(loopDelay)
	defer ticker.Stop()

	latestValues := make(map[string]int)

	for {
		select {
		case <-stopSignal:
			slog.Info("Ending sensor data generator...")
			return
		case <-ticker.C:
			for _, name := range sv.sensorNames {
				latestValues[name] = rand.Intn(1024)
			}
			sv.Update(latestValues)
		}
	}
}

func (sv *SensorViewer) setupUI() {
	sv.view = tview.NewTextView()
	sv.view.SetDynamicColors(true)
	sv.view.SetTextAlign(tview.AlignLeft)
	sv.view.SetBackgroundColor(tcell.ColorDarkSlateGray)
	sv.view.SetBorder(true).SetTitle(viewerTitle).SetTitleColor(tcell.ColorLightBlue)

	// Generate the intro text within the component
	var introText strings.Builder
	if !sv.devMode {
		introText.WriteString("Displaying real sensor values.\n")
	} else {
		introText.WriteString("[#ff0000]Caution:[-] Displaying random sensor values for development.\n")
	}
	introText.WriteString("Hit [#ff0000]q[-] to exit, [#ff0000]r[-] to reload config file and restart")

	intro := tview.NewTextView()
	intro.SetBorder(true).SetTitle(" GOLEDS Simulation ").SetTitleColor(tcell.ColorLightBlue)
	intro.SetText(introText.String())
	intro.SetTextAlign(tview.AlignCenter)
	intro.SetDynamicColors(true)
	intro.SetBackgroundColor(tcell.ColorDarkSlateGray)

	layout := tview.NewFlex().SetDirection(tview.FlexRow)
	layout.AddItem(intro, 4, 1, false)
	// The sensor view itself is 3 lines of text + 2 for the border.
	layout.AddItem(sv.view, 5, 1, true)

	// Set a reasonable overall size for the layout
	width := 22 + (colWidth * len(sv.sensorNames))
	layout.SetRect(1, 1, width, 10)

	sv.tuiApp.SetRoot(layout, true).SetFocus(sv.view)
	sv.tuiApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		key := string(event.Rune())
		switch key {
		case "q", "Q":
			sv.tuiApp.Stop()
			sv.ossignal <- os.Interrupt
		case "r", "R":
			sv.tuiApp.Stop()
			sv.ossignal <- syscall.SIGHUP
		}
		return event
	})
}

// prepareDisplayStrings generates the output strings from the current sensor data.
// This method MUST be called with the mutex already held.
func (sv *SensorViewer) prepareDisplayStrings() (string, string, string) {
	var buft, bufm, bufb strings.Builder

	// Use fmt.Sprintf with negative width for left-justified padding
	buft.WriteString(fmt.Sprintf("[yellow]%-*s[white]", colWidth+4, " [min|mean|max]"))
	bufm.WriteString(fmt.Sprintf("[yellow]%-*s[white]", colWidth+4, " Standard Deviation"))
	bufb.WriteString(fmt.Sprintf("[yellow]%-*s[white]", colWidth+4, " Name: Trigger value"))

	for _, name := range sv.sensorNames {
		cfg := sv.sensorCfgs[name]
		values, ok := sv.sensorValues[name]

		var min, max float64
		var mean, stdev float64

		if ok {
			data := make([]int, values.Len())
			for i := range values.Len() {
				data[i] = values.At(i)
			}
			stats := calculateStats(data)
			min = float64(stats.min)
			max = float64(stats.max)
			mean = math.Round(stats.mean)
			stdev = stats.stdDev
		}

		buft.WriteString(fmt.Sprintf(" [%4.0f|%4.0f|%4.0f] ", min, mean, max))
		bufm.WriteString(fmt.Sprintf("       %5.1f      ", stdev))
		bufb.WriteString(fmt.Sprintf("     [blue]%3s:[-] %-3d     ", name, cfg.TriggerValue))
	}
	return buft.String(), bufm.String(), bufb.String()
}

// draw updates the TextView with the provided strings.
// This must be called from within the TUI's main thread via QueueUpdateDraw.
func (sv *SensorViewer) draw(line1, line2, line3 string) {
	sv.view.SetText(fmt.Sprintf("%s\n%s\n%s", line1, line2, line3))
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
