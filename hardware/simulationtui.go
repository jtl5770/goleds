package hardware

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/exp/maps"
	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

var (
	// used to communicate with the TUI the display updates and the
	// keypresses (aka sensor triggers)
	content      *tview.TextView
	app          *tview.Application
	sensorline   string
	intent       string
	chartosensor map[string]string
)

func scaledColor(led p.Led) string {
	factor := 255 / math.Max(led.Red, math.Max(led.Green, led.Blue))
	red := math.Min(led.Red*factor, 255)
	green := math.Min(led.Green*factor, 255)
	blue := math.Min(led.Blue*factor, 255)
	return fmt.Sprintf("[#%02x%02x%02x]", byte(red), byte(green), byte(blue))
}

func simulateLedDisplay() {
	var buf strings.Builder
	tops := make([]string, len(SEGMENTS))
	bots := make([]string, len(SEGMENTS))
	for i, seg := range SEGMENTS {
		// led := seg.getSegmentLeds()
		tops[i], bots[i] = simulateLed(seg)
	}
	buf.WriteString(" ")
	for i := range SEGMENTS {
		buf.WriteString(tops[i])
	}
	buf.WriteString("\n ")
	for i := range SEGMENTS {
		buf.WriteString(bots[i])
	}
	buf.WriteString("\n [blue]" + sensorline + "[:]")
	content.SetText(buf.String())
}

func simulateLed(segment *Segment) (string, string) {
	if !segment.visible {
		retvalt := strings.Repeat(" ", segment.lastled-segment.firstled+1)
		retvalb := strings.Repeat("·", segment.lastled-segment.firstled+1)
		return retvalt, retvalb
	} else {
		values := segment.getSegmentLeds()
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
				} else if value == 4 {
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
				} else if value == 18 {
					buf1.WriteString("▁")
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

func InitSimulationTUI(ossignal chan os.Signal) {
	var buf strings.Builder
	buf.WriteString("Hit [blue]1[-]...[blue]" +
		fmt.Sprintf("%d", len(c.CONFIG.Hardware.Sensors.SensorCfg)) + "[-] to fire a sensor\n")
	buf.WriteString("Hit [red]q[-] to exit, [green]r[-] to reload config file and restart")

	layout := tview.NewFlex()
	layout.SetDirection(tview.FlexRow)

	intro := tview.NewTextView()
	intro.SetBorder(true).SetTitle(" GOLEDS Simulation ").SetTitleColor(tcell.ColorLightBlue)
	intro.SetText(buf.String())
	intro.SetTextAlign(3)
	intro.SetDynamicColors(true)
	intro.SetBackgroundColor(tcell.ColorBlack)

	stripe := tview.NewTextView()
	layout.AddItem(intro, 4, 1, false)
	layout.AddItem(stripe, 5, 1, false)
	layout.SetRect(1, 10, c.CONFIG.Hardware.Display.LedsTotal+4, 10)

	stripe.SetBorder(true)
	stripe.SetTextAlign(3)
	stripe.SetDynamicColors(true)
	stripe.SetBackgroundColor(tcell.ColorBlack)

	app = tview.NewApplication()
	app.SetRoot(layout, false)
	app.SetInputCapture(capture)
	stripe.SetChangedFunc(func() { app.Draw() })
	content = stripe

	chartosensor = make(map[string]string, len(Sensors))
	sensorline = strings.Repeat(" ", c.CONFIG.Hardware.Display.LedsTotal)
	sensorvals := maps.Values(Sensors)
	sort.Slice(sensorvals, func(i, j int) bool { return sensorvals[i].LedIndex < sensorvals[j].LedIndex })
	for i, sen := range sensorvals {
		index := sen.LedIndex
		sensorline = sensorline[0:index] + fmt.Sprintf("%d", i+1) + sensorline[index+1:c.CONFIG.Hardware.Display.LedsTotal]
		chartosensor[fmt.Sprintf("%d", i+1)] = sen.uid
	}

	go func() {
		app.Run()
		if intent == "quit" {
			ossignal <- os.Interrupt
		} else if intent == "restart" {
			ossignal <- syscall.SIGHUP
		}
	}()
}

func capture(event *tcell.EventKey) *tcell.EventKey {
	key := string(event.Rune())
	senuid, exist := chartosensor[key]
	if exist {
		SensorReader <- NewTrigger(senuid, 80, time.Now())
	} else if key == "q" || key == "Q" {
		intent = "quit"
		app.Stop()
	} else if key == "r" || key == "R" {
		intent = "restart"
		app.Stop()
	}
	return event
}
