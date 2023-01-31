package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	// "github.com/haxii/socks5"

	"git.tcp.direct/kayos/prox5"

	"github.com/rivo/tview"
)

var swamp *prox5.ProxyEngine

type socksLogger struct{}

var socklog = socksLogger{}

func StartUpstreamProxy(listen string) {
	if err := swamp.StartSOCKS5Server(listen, "", ""); err != nil {
		panic(err)
	}
}

func init() {
	swamp = prox5.NewProxyEngine()
	swamp.SetMaxWorkers(5)
	swamp.EnableDebug()
	swamp.SetDebugLogger(socklog)
	swamp.DisableDebugRedaction()
	// swamp.EnableDebugRedaction()
	swamp.EnableAutoScaler()
	go StartUpstreamProxy("127.0.0.1:1555")

	count := swamp.LoadProxyTXT(os.Args[1])
	if count < 1 {
		socklog.Printf("file contained no valid SOCKS host:port combos")
		os.Exit(1)
	}

	if err := swamp.Start(); err != nil {
		panic(err)
	}

	socklog.Printf("[USAGE] q: quit | d: debug | p: pause/unpause")
}

const statsFmt = ">>>>>-----<<<<<\n>>>>>Prox5<<<<<\n>>>>>-----<<<<<\n\nUptime: %s\n\nValidated: %d\nDispensed: %d\n\nMaximum Workers: %d\nActive  Workers: %d\nAsleep  Workers: %d\n\nAutoScale: %s\nSOCKS5 listening on 127.0.0.1:1555\n\n----------\n%s"

var (
	background *tview.TextView
	window     *tview.Modal
	app        *tview.Application
)

var last string

func currentString(lastMessage string) string {
	if lastMessage != last && lastMessage != "" {
		last = lastMessage
	}
	if lastMessage == "" {
		lastMessage = last
	}
	if swamp == nil {
		return ""
	}
	stats := swamp.GetStatistics()
	wMax, wRun, wIdle := swamp.GetWorkers()
	return fmt.Sprintf(statsFmt,
		stats.GetUptime().Round(time.Second), int(stats.Valid4+stats.Valid4a+stats.Valid5),
		stats.Dispensed, wMax, wRun, wIdle, swamp.GetAutoScalerStateString(), lastMessage)
}

func (s socksLogger) Errorf(format string, a ...interface{}) {
	s.Printf(format, a...)
}

func (s socksLogger) Printf(format string, a ...interface{}) {
	if app == nil {
		return
	}
	msg := fmt.Sprintf(format, a...)
	if msg == "" {
		return
	}
	app.QueueUpdateDraw(func() {
		window.SetText(currentString(msg))
	})
}

func (s socksLogger) Print(str string) {
	if app == nil {
		return
	}
	app.QueueUpdateDraw(func() {
		window.SetText(currentString(str))
	})
}

func buttons(buttonIndex int, buttonLabel string) {
	switch buttonIndex {
	case 0:
		app.Stop()
	case 1:
		if swamp.IsRunning() {
			err := swamp.Pause()
			if err != nil {
				socklog.Printf(err.Error())
			}
		} else {
			if err := swamp.Resume(); err != nil {
				socklog.Printf(err.Error())
			}
		}
	case 2:
		swamp.SetMaxWorkers(swamp.GetMaxWorkers() + 1)
	case 3:
		swamp.SetMaxWorkers(swamp.GetMaxWorkers() - 1)
	default:
		app.Stop()
	}
}

func main() {
	app = tview.NewApplication()

	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			app.QueueUpdateDraw(func() {
				window.SetText(currentString(""))
			})
			app.Sync()
		}
	}()

	window = tview.NewModal().
		SetText(currentString("Initialize")).
		AddButtons([]string{"Quit", "Pause", "+", "-"}).
		SetDoneFunc(buttons).
		SetBackgroundColor(tcell.ColorBlack).SetTextColor(tcell.ColorWhite)

	modal := func(p tview.Primitive, width, height int) tview.Primitive {
		return tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(p, height, 1, true).
				AddItem(nil, 0, 1, false), width, 1, true).
			AddItem(nil, 0, 1, false)
	}

	background = tview.NewTextView().
		SetTextColor(tcell.ColorGray).SetTextAlign(tview.AlignLeft)

	pages := tview.NewPages().
		AddPage("background", background, true, true).
		AddPage("window", modal(window, 150, 50), true, true)

	if err := app.SetRoot(pages, false).Run(); err != nil {
		panic(err)
	}
	swamp.SetDebugLogger(socklog)

	// Initialize()
}
