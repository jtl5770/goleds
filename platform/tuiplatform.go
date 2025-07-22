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
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/exp/maps"

	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/producer"
	"lautenbacher.net/goleds/util"
)

type TUIPlatform struct {
	*AbstractPlatform
	app          *tview.Application
	sensorline   string
	ledDisplay   *tview.TextView
	ossignalChan chan os.Signal
	chartosensor map[string]string
	stopChan     chan bool
}

func NewTUIPlatform(conf *config.Config, ossignalchan chan os.Signal, stopchan chan bool) *TUIPlatform {
	inst := &TUIPlatform{
		ossignalChan: ossignalchan,
		stopChan:     stopchan,
	}
	inst.AbstractPlatform = NewAbstractPlatform(conf, inst.DisplayLeds)
	return inst
}

func (s *TUIPlatform) Start() error {
	s.initSensors(s.config.Hardware.Sensors)
	s.displayManager = NewDisplayManager(s.config.Hardware.Display)
	s.initSimulationTUI(
		s.ossignalChan,
		len(s.config.Hardware.Sensors.SensorCfg),
		len(s.config.Hardware.Display.LedSegments),
		s.config.Hardware.Display.LedsTotal,
	)
	return nil
}

func (s *TUIPlatform) Stop() {
	if s.app != nil {
		s.app.Stop()
	}
}

func (s *TUIPlatform) DisplayLeds(leds []producer.Led) {
	for _, segarray := range s.displayManager.Segments {
		for _, seg := range segarray {
			seg.SetLeds(leds)
		}
	}
	s.simulateLedDisplay()
}

func (s *TUIPlatform) SensorDriver(stopSignal chan bool, wg *sync.WaitGroup) {
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

func (s *TUIPlatform) initSimulationTUI(ossignal chan os.Signal, numSensors int, numSegments int, ledsTotal int) {
	var buf strings.Builder
	buf.WriteString("Hit [blue]1[-]...[blue]" +
		fmt.Sprintf("%d", numSensors) + "[-] to fire a sensor\n")
	buf.WriteString("Hit [#ff0000]q[-] to exit, [#ff0000]r[-] to reload config file and restart")

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
	height = 8 + (2 * numSegments)
	layout.SetRect(1, 1, ledsTotal+4, 8+height)
	stripe.SetBorder(true)
	stripe.SetTextAlign(0)
	stripe.SetDynamicColors(true)
	stripe.SetBackgroundColor(tcell.ColorDarkSlateGray)

	s.app = tview.NewApplication()
	s.app.SetRoot(layout, false)
	s.app.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			key := string(event.Rune())
			senuid, exist := s.chartosensor[key]
			if exist {
				s.sensorEvents <- util.NewTrigger(senuid, 80, time.Now())
			} else if key == "q" || key == "Q" {
				s.app.Stop()
				ossignal <- os.Interrupt
			} else if key == "r" || key == "R" {
				s.app.Stop()
				ossignal <- syscall.SIGHUP
			}
			return event
		})
	stripe.SetChangedFunc(func() { s.app.Draw() })
	s.ledDisplay = stripe

	s.chartosensor = make(map[string]string, len(s.sensors))
	s.sensorline = strings.Repeat(" ", ledsTotal)
	sensorvals := maps.Values(s.sensors)
	sort.Slice(sensorvals, func(i, j int) bool { return sensorvals[i].LedIndex < sensorvals[j].LedIndex })
	for i, sen := range sensorvals {
		index := sen.LedIndex
		s.sensorline = s.sensorline[0:index] + fmt.Sprintf("%d", i+1) + s.sensorline[index+1:ledsTotal]
		s.chartosensor[fmt.Sprintf("%d", i+1)] = sen.uid
	}

	go func() {
		if err := s.app.Run(); err != nil {
			log.Fatalf("Error running TUI: %v", err)
		}
	}()
}

func (s *TUIPlatform) simulateLedDisplay() {
	var buf strings.Builder
	keys := maps.Keys(s.displayManager.Segments)
	sort.Strings(keys)
	for _, name := range keys {
		segarray := s.displayManager.Segments[name]
		tops := make([]string, len(segarray))
		bots := make([]string, len(segarray))
		for i, seg := range segarray {
			tops[i], bots[i] = s.simulateLed(seg)
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
	buf.WriteString(" [blue]" + s.sensorline + "[:]")
	s.ledDisplay.SetText(buf.String())
}

func (s *TUIPlatform) simulateLed(segment *Segment) (string, string) {
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
