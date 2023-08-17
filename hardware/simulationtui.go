package hardware

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

var (
	// used to communicate with the TUI the display updates and the
	// keypresses (aka sensor triggers)
	CONTENT *tview.TextView
	KEYCHAN chan Trigger
)

func scaledColor(led p.Led) string {
	factor := 255 / math.Max(led.Red, math.Max(led.Green, led.Blue))
	red := math.Min(led.Red*factor, 255)
	green := math.Min(led.Green*factor, 255)
	blue := math.Min(led.Blue*factor, 255)
	color := fmt.Sprintf("[#%02x%02x%02x]", byte(red), byte(green), byte(blue))
	return color
}

func simulateLedDisplay(led1 []p.Led, led2 []p.Led) {
	var buf strings.Builder
	buf.WriteString(" ① ")
	buf.WriteString(simulateLed(0, led1))
	buf.WriteString(" ② ······· ③ ")
	buf.WriteString(simulateLed(1, led2))
	buf.WriteString(" ④ ")
	CONTENT.SetText(buf.String())
}

func simulateLed(segmentID int, values []p.Led) string {
	var buf strings.Builder
	buf.Grow(len(values))
	for _, v := range values {
		if v.IsEmpty() {
			buf.WriteString(" ")
		} else {
			value := byte(math.Round(float64(v.Red+v.Green+v.Blue) / 3.0))
			buf.WriteString(scaledColor(v))
			if value == 1 {
				buf.WriteString("▁")
			} else if value == 2 {
				buf.WriteString("▂")
			} else if value <= 4 {
				buf.WriteString("▃")
			} else if value <= 8 {
				buf.WriteString("▄")
			} else if value <= 16 {
				buf.WriteString("▅")
			} else if value <= 24 {
				buf.WriteString("▆")
			} else if value <= 32 {
				buf.WriteString("▇")
			} else {
				buf.WriteString("█")
			}
			buf.WriteString("[-]")
		}
	}
	return buf.String()
}

// I obviously have no clue what I am doing here
func SetupDebugUI() {
	var buf strings.Builder
	buf.WriteString("Hit [blue]1[-],[blue]2[-],[blue]3[-] or [blue]4[-] to fire a sensor\n")
	buf.WriteString("Hit [red]Ctrl-C[-] to drop back to the terminal")

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
	layout.AddItem(stripe, 3, 1, false)
	layout.SetRect(1, 10, c.CONFIG.Hardware.Display.LedsTotal+21, 10)

	stripe.SetBorder(true)
	stripe.SetTextAlign(3)
	stripe.SetDynamicColors(true)
	stripe.SetBackgroundColor(tcell.ColorBlack)

	app := tview.NewApplication()
	app.SetRoot(layout, false)
	app.SetInputCapture(capture)
	stripe.SetChangedFunc(func() { app.Draw() })
	CONTENT = stripe
	go func() {
		defer os.Exit(0)
		app.Run()
	}()
}

func capture(event *tcell.EventKey) *tcell.EventKey {
	key := event.Rune()
	if key == '1' {
		KEYCHAN <- Trigger{"S0", 80, time.Now()}
	} else if key == '2' {
		KEYCHAN <- Trigger{"S1", 80, time.Now()}
	} else if key == '3' {
		KEYCHAN <- Trigger{"S2", 80, time.Now()}
	} else if key == '4' {
		KEYCHAN <- Trigger{"S3", 80, time.Now()}
	}
	return event
}