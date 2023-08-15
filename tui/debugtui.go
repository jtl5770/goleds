package tui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	c "lautenbacher.net/goleds/config"
)

var (
	CONTENT *tview.TextView
	KEYCHAN chan rune
)

// I obviously have no clue what I am doing here
func SetupDebugUI() {
	var buf strings.Builder
	buf.WriteString("Enter [blue]1[-],[blue]2[-],[blue]3[-] or [blue]4[-] to fire a sensor\n")
	buf.WriteString("Enter [blue:b]Ctrl-C[-] to drop back to the terminal, repeat to quit completely")

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
	stripe.SetText("This is the [red]CONTENT[-] to be displayed")

	app := tview.NewApplication()
	app.SetRoot(layout, false)
	app.SetInputCapture(capture)
	go app.Run()
	stripe.SetChangedFunc(func() { app.Draw() })
	CONTENT = stripe
}

func capture(event *tcell.EventKey) *tcell.EventKey {
	key := event.Rune()
	KEYCHAN <- key
	return event
}
