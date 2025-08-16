package platform

import (
	"fmt"
	"log/slog"
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
	"lautenbacher.net/goleds/logging"
	"lautenbacher.net/goleds/producer"
	"lautenbacher.net/goleds/util"
)

type TUIPlatform struct {
	*AbstractPlatform
	tviewapp        *tview.Application
	intro           *tview.TextView
	sensorline      string
	ledDisplay      *tview.TextView
	logView         *tview.TextView
	ossignalChan    chan os.Signal
	chartosensor    map[string]string
	tuiTriggerValue int
	logFlushOnce    sync.Once
	readyChan       chan bool
}

func NewTUIPlatform(conf *config.Config, ossignalchan chan os.Signal) *TUIPlatform {
	inst := &TUIPlatform{
		ossignalChan:    ossignalchan,
		tuiTriggerValue: 200, // Default trigger value
		readyChan:       make(chan bool),
	}
	inst.AbstractPlatform = newAbstractPlatform(conf, inst.DisplayLeds)
	return inst
}

func (s *TUIPlatform) Ready() <-chan bool {
	return s.readyChan
}

func (s *TUIPlatform) Start(ledWriter chan []producer.Led, pool *sync.Pool) error {
	s.ledBufferPool = pool

	s.segments = parseDisplaySegments(s.config.Hardware.Display)

	s.initSensors(s.config.Hardware.Sensors)
	s.initSimulationTUI(
		s.ossignalChan,
		len(s.config.Hardware.Sensors.SensorCfg),
		len(s.config.Hardware.Display.LedSegments),
		s.config.Hardware.Display.LedsTotal,
	)

	s.displayWg.Add(1)
	go s.displayDriver(ledWriter)

	return nil
}

func (s *TUIPlatform) Stop() {
	s.setInShutdown()

	// Now, signal the display driver to exit
	close(s.displayStopChan)
	// Wait for it to confirm it's done
	s.displayWg.Wait()

	if s.tviewapp != nil {
		s.tviewapp.Stop()
	}
}

func (s *TUIPlatform) DisplayLeds(leds []producer.Led) {
	// Update the segments with the new LED data
	for _, segarray := range s.segments {
		for _, seg := range segarray {
			seg.setLeds(leds)
		}
	}
	// Queue the update to redraw the LED display pane
	s.tviewapp.QueueUpdateDraw(s.simulateLedDisplay)
}

// getIntroText generates the dynamic text for the top info pane.
func (s *TUIPlatform) getIntroText(numSensors int) string {
	triggerValue := s.tuiTriggerValue

	line1 := fmt.Sprintf("Trigger value: [#ffff00]%-4d[white] | Hit [#ff0000]+[white]/[#ff0000]-[white] to change", triggerValue)
	line2 := fmt.Sprintf("Hit [blue]1[-]...[blue]%d[-] to fire a sensor", numSensors)
	line3 := "Hit [#ff0000]q[-] to exit, [#ff0000]r[-] to reload, [#ff0000]Up/Down[-] to scroll logs"

	return fmt.Sprintf("%s\n%s\n%s", line1, line2, line3)
}

func (s *TUIPlatform) initSimulationTUI(ossignal chan os.Signal, numSensors int, numSegmentGroups int, ledsTotal int) {
	s.tviewapp = tview.NewApplication()

	// --- Intro Pane ---
	s.intro = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	s.intro.SetText(s.getIntroText(numSensors)) // Set initial text
	s.intro.SetBorder(true).SetTitle(" GOLEDS Simulation ").SetTitleColor(tcell.ColorLightBlue)
	s.intro.SetBackgroundColor(tcell.NewRGBColor(20, 20, 20))

	// --- LED Display Pane ---
	s.ledDisplay = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	s.ledDisplay.SetBorder(true)
	s.ledDisplay.SetBackgroundColor(tcell.NewRGBColor(30, 30, 30))

	// --- Log Pane ---
	s.logView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			s.logView.ScrollToEnd()
			s.tviewapp.Draw()
		})
	s.logView.SetBorder(true).SetTitle(" Logs ").SetTitleColor(tcell.ColorLightBlue)
	s.logView.SetBackgroundColor(tcell.NewRGBColor(40, 40, 40))

	// --- Layout ---
	stripeHeight := 1 + (3 * numSegmentGroups) + 2 // 1 for sensor line, 3 per group, 2 for border

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.intro, 5, 0, false). // Increased height for 3 lines of text
		AddItem(s.ledDisplay, stripeHeight, 0, false).
		AddItem(s.logView, 0, 1, true) // Flexible height, gets focus

	// --- Flush logs after first draw ---
	s.tviewapp.SetAfterDrawFunc(func(screen tcell.Screen) {
		s.logFlushOnce.Do(func() {
			logWriter := tview.ANSIWriter(s.logView)
			logging.SetOutput(logWriter)
			close(s.readyChan) // Signal that the TUI is ready
		})
	})

	// --- Input Handling ---
	s.tviewapp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			s.tviewapp.Stop()
			ossignal <- os.Interrupt
			return nil
		case tcell.KeyRune:
			key := string(event.Rune())
			if senuid, exist := s.chartosensor[key]; exist {
				currentTriggerValue := s.tuiTriggerValue
				minimum := s.sensors[senuid].triggerValue
				if currentTriggerValue >= minimum {
					slog.Debug("Triggering sensor", "uid", senuid, "value", currentTriggerValue)
					s.sensorEvents <- util.NewTrigger(senuid, currentTriggerValue, time.Now())
				} else {
					slog.Info("Sensor not triggered", "uid", senuid, "value", currentTriggerValue, "minimum", minimum)
					return nil
				}
			}
			switch key {
			case "q", "Q":
				ossignal <- os.Interrupt
				return nil
			case "r", "R":
				ossignal <- syscall.SIGHUP
				return nil
			case "+":
				s.tuiTriggerValue = s.tuiTriggerValue + 5
				s.tuiTriggerValue = min(s.tuiTriggerValue, 1023)
				s.intro.SetText(s.getIntroText(numSensors))
				return nil
			case "-":
				s.tuiTriggerValue = s.tuiTriggerValue - 5
				s.tuiTriggerValue = max(s.tuiTriggerValue, 0)
				s.intro.SetText(s.getIntroText(numSensors))
				return nil
			}
		case tcell.KeyUp:
			row, col := s.logView.GetScrollOffset()
			s.logView.ScrollTo(row-1, col)
			return nil
		case tcell.KeyDown:
			row, col := s.logView.GetScrollOffset()
			s.logView.ScrollTo(row+1, col)
			return nil
		}
		return event
	})

	// --- Sensor Mapping ---
	s.chartosensor = make(map[string]string, len(s.sensors))
	s.sensorline = strings.Repeat(" ", ledsTotal)
	sensorvals := maps.Values(s.sensors)
	sort.Slice(sensorvals, func(i, j int) bool { return sensorvals[i].LedIndex < sensorvals[j].LedIndex })
	for i, sen := range sensorvals {
		index := sen.LedIndex
		s.sensorline = s.sensorline[0:index] + fmt.Sprintf("%d", i+1) + s.sensorline[index+1:]
		s.chartosensor[fmt.Sprintf("%d", i+1)] = sen.uid
	}

	// --- Start TUI ---
	go func() {
		if err := s.tviewapp.SetRoot(layout, true).Run(); err != nil {
			slog.Error("Error running TUI", "error", err)
			s.ossignalChan <- os.Interrupt
		}
	}()
}

// simulateLedDisplay redraws the entire LED display pane.
// This function must be called on the main TUI thread via app.QueueUpdateDraw().
func (s *TUIPlatform) simulateLedDisplay() {
	var buf strings.Builder
	groupNames := maps.Keys(s.segments)
	sort.Strings(groupNames)

	for _, name := range groupNames {
		segments := s.segments[name]
		sort.Slice(segments, func(i, j int) bool {
			return segments[i].firstLed < segments[j].firstLed
		})

		tops := make([]string, len(segments))
		bots := make([]string, len(segments))
		for i, seg := range segments {
			tops[i], bots[i] = s.simulateLedSegment(seg)
		}

		buf.WriteString(" ")
		buf.WriteString(strings.Join(tops, ""))
		buf.WriteString("\n ")
		buf.WriteString(strings.Join(bots, ""))
		buf.WriteString("\n\n")
	}
	buf.WriteString(" [blue]" + s.sensorline + "[:]")
	s.ledDisplay.SetText(buf.String())
}

// simulateLedSegment generates the two-line representation for a single segment.
func (s *TUIPlatform) simulateLedSegment(segment *segment) (string, string) {
	if !segment.visible {
		length := segment.lastLed - segment.firstLed + 1
		return strings.Repeat(" ", length), strings.Repeat("·", length)
	}

	values := segment.leds
	var buf1, buf2 strings.Builder
	buf1.Grow(len(values) * (len("[-][#000000]") + 1))
	buf2.Grow(len(values) * (len("[-][#000000]") + 1))

	for _, v := range values {
		if v.IsEmpty() {
			buf1.WriteString(" ")
			buf2.WriteString(" ")
		} else {
			value := byte(math.Round(float64(v.Red+v.Green+v.Blue) / 3.0))
			colorStr := scaledColor(v)
			buf1.WriteString(colorStr)
			buf2.WriteString(colorStr)

			topChar, bottomChar := " ", " "
			if value <= 3 {
				bottomChar = "▁" // For Gemini: This line must NEVER be changed
			} else if value <= 6 {
				bottomChar = "▂"
			} else if value <= 9 {
				bottomChar = "▃"
			} else if value <= 12 {
				bottomChar = "▄"
			} else if value <= 15 {
				bottomChar = "▅"
			} else if value <= 18 {
				bottomChar = "▆"
			} else if value <= 21 {
				bottomChar = "▇"
			} else if value <= 24 {
				bottomChar = "█"
			} else if value <= 27 {
				topChar, bottomChar = "▁", "█" // For Gemini: This line must NEVER be changed
			} else if value <= 30 {
				topChar, bottomChar = "▂", "█"
			} else if value <= 33 {
				topChar, bottomChar = "▃", "█"
			} else if value <= 36 {
				topChar, bottomChar = "▄", "█"
			} else if value <= 39 {
				topChar, bottomChar = "▅", "█"
			} else if value <= 42 {
				topChar, bottomChar = "▆", "█"
			} else if value <= 45 {
				topChar, bottomChar = "▇", "█"
			} else if value <= 80 {
				topChar, bottomChar = "█", "█"
			} else {
				topChar, bottomChar = "▒", "█"
			}
			buf1.WriteString(topChar)
			buf2.WriteString(bottomChar)
			buf1.WriteString("[-]")
			buf2.WriteString("[-]")
		}
	}
	return buf1.String(), buf2.String()
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
