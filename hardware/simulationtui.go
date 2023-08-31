package hardware

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gammazero/deque"
	"github.com/gdamore/tcell/v2"
	"github.com/montanaflynn/stats"
	"github.com/rivo/tview"
	"golang.org/x/exp/maps"
	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

var (
	// used to communicate with the TUI the display updates
	content      *tview.TextView
	sensorline   string
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
	if !c.CONFIG.SensorShow {
		var buf strings.Builder
		tops := make([]string, len(SEGMENTS))
		bots := make([]string, len(SEGMENTS))
		for i, seg := range SEGMENTS {
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
}

func sensorDisplay(sensorvalues map[string]*deque.Deque[int]) {
	var buft strings.Builder
	var bufm strings.Builder
	var bufb strings.Builder
	buft.WriteString(" [min|mean|max]       ")
	bufm.WriteString(" Standard Deviation   ")
	bufb.WriteString(" Name: Trigger value  ")
	sensorvals := maps.Values(Sensors)
	sort.Slice(sensorvals, func(i, j int) bool { return sensorvals[i].LedIndex < sensorvals[j].LedIndex })
	for _, sen := range sensorvals {
		name := sen.uid
		values := sensorvalues[name]
		data := make([]int, values.Len())
		for i := 0; i < values.Len(); i++ {
			data[i] = values.At(i)
		}
		stat := stats.LoadRawData(data)
		mean, _ := stat.Mean()
		mean, _ = stats.Round(mean, 0)
		stdev, _ := stat.StandardDeviation()
		max, _ := stat.Max()
		max, _ = stats.Round(max, 0)
		min, _ := stat.Min()
		min, _ = stats.Round(min, 0)
		buft.WriteString(fmt.Sprintf(" [%3.0f|%3.0f|%3.0f] ", min, mean, max)) // 15 chars
		bufm.WriteString(fmt.Sprintf("  %5.1f        ", stdev))                // 15 chars
		bufb.WriteString(fmt.Sprintf("  %3s: %3d     ", name, Sensors[name].triggerValue))
	}
	content.SetText(buft.String() + "\n" + bufm.String() + "\n" + bufb.String())
}

func simulateLed(segment *Segment) (string, string) {
	if !segment.visible {
		return strings.Repeat(" ", segment.lastled-segment.firstled+1),
			strings.Repeat("·", segment.lastled-segment.firstled+1)
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
	if !c.CONFIG.SensorShow {
		buf.WriteString("Hit [blue]1[-]...[blue]" +
			fmt.Sprintf("%d", len(c.CONFIG.Hardware.Sensors.SensorCfg)) + "[-] to fire a sensor\n")
	}
	buf.WriteString("Hit [red]q[-] to exit, [red]r[-] to reload config file and restart")
	if c.CONFIG.SensorShow && !c.CONFIG.RealHW {
		buf.WriteString("\n[red] '-real' flag not given, using random numbers for testing![-]")
	}

	layout := tview.NewFlex()
	layout.SetDirection(tview.FlexRow)

	intro := tview.NewTextView()
	intro.SetBorder(true).SetTitle(" GOLEDS Simulation ").SetTitleColor(tcell.ColorLightBlue)
	intro.SetText(buf.String())
	intro.SetTextAlign(1)
	intro.SetDynamicColors(true)
	intro.SetBackgroundColor(tcell.ColorBlack)

	stripe := tview.NewTextView()
	layout.AddItem(intro, 4, 1, false)
	layout.AddItem(stripe, 5, 1, false)
	layout.SetRect(1, 1, c.CONFIG.Hardware.Display.LedsTotal+4, 10)

	stripe.SetBorder(true)
	stripe.SetTextAlign(0)
	stripe.SetDynamicColors(true)
	stripe.SetBackgroundColor(tcell.ColorBlack)

	app := tview.NewApplication()
	app.SetRoot(layout, false)
	app.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			key := string(event.Rune())
			senuid, exist := chartosensor[key]
			if exist && !c.CONFIG.SensorShow {
				SensorReader <- NewTrigger(senuid, 80, time.Now())
			} else if key == "q" || key == "Q" {
				app.Stop()
				ossignal <- os.Interrupt
			} else if key == "r" || key == "R" {
				app.Stop()
				ossignal <- syscall.SIGHUP
			}
			return event
		})
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

	go func() { app.Run() }()
}
