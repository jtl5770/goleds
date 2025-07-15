package tui

import (
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/exp/maps"

	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/platform"
	"lautenbacher.net/goleds/producer"
)

type TUIPlatform struct {
	app            *tview.Application
	ledDisplay     *tview.TextView
	sensorEvents   chan *platform.Trigger
	osSignalChan   chan os.Signal
	config         *config.Config
	displayManager *platform.DisplayManager
	sensorline     string
	chartosensor   map[string]string
	sensors        map[string]*sensor // Keep sensor map for internal use
	stopChan       chan bool
}

// sensor struct and related functions (now internal to TUIPlatform)
type sensor struct {
	uid          string
	LedIndex     int
	spimultiplex string
	adcChannel   byte
	triggerValue int
	values       []int
	smoothing    int
}

func (p *TUIPlatform) initSensors(sensorConfig config.SensorsConfig) {
	p.sensors = make(map[string]*sensor, len(sensorConfig.SensorCfg))
	for uid, cfg := range sensorConfig.SensorCfg {
		p.sensors[uid] = p.newSensor(uid, cfg.LedIndex, cfg.SpiMultiplex, cfg.AdcChannel, cfg.TriggerValue, sensorConfig.SmoothingSize)
	}
}

func (p *TUIPlatform) newSensor(uid string, ledIndex int, spimultiplex string, adcChannel byte, triggerValue int, smoothing int) *sensor {
	return &sensor{
		uid:          uid,
		LedIndex:     ledIndex,
		spimultiplex: spimultiplex,
		adcChannel:   adcChannel,
		triggerValue: triggerValue,
		values:       make([]int, smoothing, smoothing+1),
		smoothing:    smoothing,
	}
}

func (s *sensor) smoothValue(val int) int {
	var ret int
	newValues := make([]int, s.smoothing, s.smoothing+1)
	for index, curr := range append(s.values, val)[1:] {
		newValues[index] = curr
		ret += curr
	}
	s.values = newValues
	return ret / s.smoothing
}

func NewPlatform(osSignalChan chan os.Signal, conf *config.Config) *TUIPlatform {
	return &TUIPlatform{
		osSignalChan: osSignalChan,
		config:       conf,
		sensorEvents: make(chan *platform.Trigger),
		stopChan:     make(chan bool),
	}
}

func (p *TUIPlatform) Start() error {
	p.initSensors(p.config.Hardware.Sensors)
	p.displayManager = platform.NewDisplayManager(p.config.Hardware.Display)
	p.initSimulationTUI(
		p.osSignalChan,
		p.config.SensorShow,
		p.config.RealHW,
		len(p.config.Hardware.Sensors.SensorCfg),
		len(p.config.Hardware.Display.LedSegments),
		p.config.Hardware.Display.LedsTotal,
	)
	return nil
}

func (p *TUIPlatform) Stop() {
	if p.app != nil {
		p.app.Stop()
	}
	close(p.stopChan)
}

func (p *TUIPlatform) DisplayLeds(leds []producer.Led) {
	for _, segarray := range p.displayManager.Segments {
		for _, seg := range segarray {
			seg.SetLeds(leds)
		}
	}
	p.simulateLedDisplay(p.config.SensorShow)
}

func (p *TUIPlatform) GetSensorEvents() <-chan *platform.Trigger {
	return p.sensorEvents
}

func (p *TUIPlatform) GetSensorLedIndices() map[string]int {
	indices := make(map[string]int)
	for uid, sensor := range p.sensors {
		indices[uid] = sensor.LedIndex
	}
	return indices
}

func (p *TUIPlatform) GetLedsTotal() int {
	return p.config.Hardware.Display.LedsTotal
}

func (p *TUIPlatform) GetForceUpdateDelay() time.Duration {
	return p.config.Hardware.Display.ForceUpdateDelay
}

func (p *TUIPlatform) DisplayDriver(display chan []producer.Led, stopSignal chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-stopSignal:
			log.Println("Ending DisplayDriver go-routine (TUI)")
			return
		case sumLeds := <-display:
			p.DisplayLeds(sumLeds)
		}
	}
}

func (p *TUIPlatform) SensorDriver(stopSignal chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	// In the TUI platform, sensor events are triggered by key presses,
	// not by a continuous reading loop. This function is here to satisfy the
	// platform.Platform interface, but it doesn't need to do anything.
	for {
		select {
		case <-stopSignal:
			log.Println("Ending SensorDriver go-routine (TUI)")
			return
		}
	}
}

func (p *TUIPlatform) initSimulationTUI(ossignal chan os.Signal, sensorShow bool, realHW bool, numSensors int, numSegments int, ledsTotal int) {
	var buf strings.Builder
	if !sensorShow {
		buf.WriteString("Hit [blue]1[-]...[blue]" +
			fmt.Sprintf("%d", numSensors) + "[-] to fire a sensor\n")
	}
	buf.WriteString("Hit [#ff0000]q[-] to exit, [#ff0000]r[-] to reload config file and restart")
	if sensorShow && !realHW {
		buf.WriteString("\n[#ff0000] '-real' flag not given, using random numbers for testing![-]")
	}

	layout := tview.NewFlex()
	layout.SetDirection(tview.FlexRow)

	intro := tview.NewTextView()
	intro.SetBorder(true).SetTitle(" GOLEDS Simulation ").SetTitleColor(tcell.ColorLightBlue)
	intro.SetText(buf.String())
	intro.SetTextAlign(1)
	intro.SetDynamicColors(true)
	intro.SetBackgroundColor(tcell.ColorDarkSlateGray)

	stripe := tview.NewTextView()
	height := 3 * numSegments
	layout.AddItem(intro, 4, 1, false)
	layout.AddItem(stripe, 3+height, 1, false)
	if sensorShow {
		layout.SetRect(1, 1, int(math.Max(float64(numSensors*15+24), 70)), 10)
	} else {
		height := 8 + (2 * numSegments)
		layout.SetRect(1, 1, ledsTotal+4, 8+height)
	}
	stripe.SetBorder(true)
	stripe.SetTextAlign(0)
	stripe.SetDynamicColors(true)
	stripe.SetBackgroundColor(tcell.ColorDarkSlateGray)

	p.app = tview.NewApplication()
	p.app.SetRoot(layout, false)
	p.app.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			key := string(event.Rune())
			senuid, exist := p.chartosensor[key]
			if exist && !sensorShow {
				p.sensorEvents <- platform.NewTrigger(senuid, 80, time.Now())
			} else if key == "q" || key == "Q" {
				p.app.Stop()
				ossignal <- os.Interrupt
			} else if key == "r" || key == "R" {
				p.app.Stop()
				ossignal <- syscall.SIGHUP
			}
			return event
		})
	stripe.SetChangedFunc(func() { p.app.Draw() })
	p.ledDisplay = stripe

	p.chartosensor = make(map[string]string, len(p.sensors))
	p.sensorline = strings.Repeat(" ", ledsTotal)
	sensorvals := maps.Values(p.sensors)
	sort.Slice(sensorvals, func(i, j int) bool { return sensorvals[i].LedIndex < sensorvals[j].LedIndex })
	for i, sen := range sensorvals {
		index := sen.LedIndex
		p.sensorline = p.sensorline[0:index] + fmt.Sprintf("%d", i+1) + p.sensorline[index+1:ledsTotal]
		p.chartosensor[fmt.Sprintf("%d", i+1)] = sen.uid
	}

	go func() {
		if err := p.app.Run(); err != nil {
			log.Fatalf("Error running TUI: %v", err)
		}
	}()
}

func (p *TUIPlatform) simulateLedDisplay(sensorShow bool) {
	if p.ledDisplay == nil {
		return
	}
	if !sensorShow {
		var buf strings.Builder
		keys := maps.Keys(p.displayManager.Segments)
		sort.Strings(keys)
		for _, name := range keys {
			segarray := p.displayManager.Segments[name]
			tops := make([]string, len(segarray))
			bots := make([]string, len(segarray))
			for i, seg := range segarray {
				tops[i], bots[i] = p.simulateLed(seg)
			}
			buf.WriteString(" ")
			for i := range segarray {
				buf.WriteString(tops[i])
			}
			buf.WriteString("\n ")
			for i := range segarray {
				buf.WriteString(bots[i])
			}
			buf.WriteString("\n\n")
		}
		buf.WriteString(" [blue]" + p.sensorline + "[:]")
		p.ledDisplay.SetText(buf.String())
	}
}

func (p *TUIPlatform) simulateLed(segment *platform.Segment) (string, string) {
	if !segment.Visible {
		return strings.Repeat(" ", segment.LastLed-segment.FirstLed+1),
			strings.Repeat("·", segment.LastLed-segment.FirstLed+1)
	} else {
		values := segment.Leds
		var buf1 strings.Builder
		var buf2 strings.Builder
		buf1.Grow(len(values))
		buf2.Grow(len(values))
		for _, v := range values {
			if v.IsEmpty() {
				buf1.WriteString(" ")
				buf2.WriteString(" ")
			} else {
				value := byte(math.Round(float64(v.Red+v.Green+v.Blue) / 3.0))
				buf1.WriteString(scaledColor(v))
				buf2.WriteString(scaledColor(v))
				if value <= 2 {
					buf1.WriteString(" ")
					buf2.WriteString("▁")
				} else if value <= 4 {
					buf1.WriteString(" ")
					buf2.WriteString("▂")
				} else if value <= 6 {
					buf1.WriteString(" ")
					buf2.WriteString("▃")
				} else if value <= 8 {
					buf1.WriteString(" ")
					buf2.WriteString("▄")
				} else if value <= 10 {
					buf1.WriteString(" ")
					buf2.WriteString("▅")
				} else if value <= 12 {
					buf1.WriteString(" ")
					buf2.WriteString("▆")
				} else if value <= 14 {
					buf1.WriteString(" ")
					buf2.WriteString("▇")
				} else if value <= 16 {
					buf1.WriteString(" ")
					buf2.WriteString("█")
				} else if value <= 18 {
					buf1.WriteString(" ")
					buf2.WriteString("█")
				} else if value <= 20 {
					buf1.WriteString("▂")
					buf2.WriteString("█")
				} else if value <= 22 {
					buf1.WriteString("▃")
					buf2.WriteString("█")
				} else if value <= 24 {
					buf1.WriteString("▄")
					buf2.WriteString("█")
				} else if value <= 26 {
					buf1.WriteString("▅")
					buf2.WriteString("█")
				} else if value <= 28 {
					buf1.WriteString("▆")
					buf2.WriteString("█")
				} else if value <= 30 {
					buf1.WriteString("▇")
					buf2.WriteString("█")
				} else {
					buf1.WriteString("█")
					buf2.WriteString("█")
				}
				buf1.WriteString("[-]")
				buf2.WriteString("[-]")
			}
		}
		return buf1.String(), buf2.String()
	}
}

func scaledColor(led producer.Led) string {
	maxColor := math.Max(led.Red, math.Max(led.Green, led.Blue))
	if maxColor == 0 {
		return "[#000000]"
	}
	factor := 255 / maxColor
	red := math.Min(led.Red*factor, 255)
	green := math.Min(led.Green*factor, 255)
	blue := math.Min(led.Blue*factor, 255)

	const epsilon = 1e-9

	return fmt.Sprintf("[#%02x%02x%02x]", byte(math.Round(red+epsilon)), byte(math.Round(green+epsilon)), byte(math.Round(blue+epsilon)))
}
