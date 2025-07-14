package driver

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
	"golang.org/x/exp/slices"
	p "lautenbacher.net/goleds/producer"
)

var (
	// used to communicate with the TUI the display updates
	content      *tview.TextView
	sensorline   string
	chartosensor map[string]string
)

func scaledColor(led p.Led) string {
	maxColor := math.Max(led.Red, math.Max(led.Green, led.Blue))
	if maxColor == 0 {
		return "[#000000]"
	}
	factor := 255 / maxColor
	red := math.Min(led.Red*factor, 255)
	green := math.Min(led.Green*factor, 255)
	blue := math.Min(led.Blue*factor, 255)

	// Add a small epsilon to counter floating-point inaccuracies before rounding.
	// This helps cases like 127.5 being represented as 127.499... and rounding down.
	const epsilon = 1e-9

	return fmt.Sprintf("[#%02x%02x%02x]", byte(math.Round(red+epsilon)), byte(math.Round(green+epsilon)), byte(math.Round(blue+epsilon)))
}

func simulateLedDisplay(sensorShow bool) {
	if !sensorShow {
		var buf strings.Builder
		keys := maps.Keys(SEGMENTS)
		slices.Sort(keys)
		for _, name := range keys {
			segarray := SEGMENTS[name]
			tops := make([]string, len(segarray))
			bots := make([]string, len(segarray))
			for i, seg := range segarray {
				tops[i], bots[i] = simulateLed(seg)
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
		buf.WriteString(" [blue]" + sensorline + "[:]")
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

func simulateLed(segment *ledsegment) (string, string) {
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

func InitSimulationTUI(ossignal chan os.Signal, sensorShow bool, realHW bool, numSensors int, numSegments int, ledsTotal int) {
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

	app := tview.NewApplication()
	app.SetRoot(layout, false)
	app.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			key := string(event.Rune())
			senuid, exist := chartosensor[key]
			if exist && !sensorShow {
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
	sensorline = strings.Repeat(" ", ledsTotal)
	sensorvals := maps.Values(Sensors)
	sort.Slice(sensorvals, func(i, j int) bool { return sensorvals[i].LedIndex < sensorvals[j].LedIndex })
	for i, sen := range sensorvals {
		index := sen.LedIndex
		sensorline = sensorline[0:index] + fmt.Sprintf("%d", i+1) + sensorline[index+1:ledsTotal]
		chartosensor[fmt.Sprintf("%d", i+1)] = sen.uid
	}

	go func() { app.Run() }()
}
