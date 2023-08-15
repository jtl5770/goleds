package tui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	CONTENT *tview.TextView
	KEYCHAN chan rune
)

func SetupDebugUI() {
	var buf strings.Builder
	buf.WriteString("Enter [blue]1[-],[blue]2[-],[blue]3[-] or [blue]4[-] to fire a sensor\n")
	buf.WriteString("Enter [blue:b]Ctrl-C[-] 2 times to quit")

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
	layout.AddItem(stripe, 1, 1, false)
	layout.SetRect(0, 10, 150, 10)

	stripe.SetBorder(false)
	stripe.SetTextAlign(3)
	stripe.SetDynamicColors(true)
	stripe.SetBackgroundColor(tcell.ColorBlack)
	stripe.SetText("This is the [red]CONTENT[-] to be displayed")
	app := tview.NewApplication()
	app.SetRoot(layout, false)
	app.SetFocus(layout)
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
