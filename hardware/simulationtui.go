package hardware

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

var (
	CONTENT *tview.TextView
	KEYCHAN chan Trigger

	// magic numbers to account for different intensities of color
	// components in led stripe to get a warm white. Needed because
	// terminal output doesn't have such a huge color cast
	magic_factor_green float64 = 5.7
	magic_factor_blue  float64 = 28.3
)

func scaledColor(led p.Led) string {
	var factor float64
	red := float64(led.Red)
	green := math.Min(float64(led.Green)*magic_factor_green, 255)
	blue := math.Min(float64(led.Blue)*magic_factor_blue, 255)

	factor = float64(255 / math.Max(red, math.Max(green, blue)))
	red = math.Min(red*factor, 255)
	green = math.Min(green*factor, 255)
	blue = math.Min(blue*factor, 255)
	color := fmt.Sprintf("[#%02x%02x%02x]", byte(red), byte(green), byte(blue))
	return color
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
	buf.WriteString("Enter [blue]1[-],[blue]2[-],[blue]3[-] or [blue]4[-] to fire a sensor\n")
	buf.WriteString("Enter [red]Ctrl-C[-] to drop back to the terminal, repeat to quit completely")

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
	layout.SetRect(2, 10, c.CONFIG.Hardware.Display.LedsTotal+20, 10)

	stripe.SetBorder(true)
	stripe.SetTextAlign(3)
	stripe.SetDynamicColors(true)
	stripe.SetBackgroundColor(tcell.ColorBlack)

	app := tview.NewApplication()
	app.SetRoot(layout, false)
	app.SetInputCapture(capture)
	go app.Run()
	stripe.SetChangedFunc(func() { app.Draw() })
	CONTENT = stripe
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
