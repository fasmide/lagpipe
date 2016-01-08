package main

import (
	//"net"

	ui "github.com/gizak/termui"
)

type Monitor struct {
	config *MonitorConfig
}

func NewMonitor(c *MonitorConfig) Monitor {
	return Monitor{config: c}
}

func (m *Monitor) Start() {
	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

	help := ui.NewPar(":PRESS q TO QUIT")
	help.Height = 3
	help.Width = 50
	help.TextFgColor = ui.ColorWhite
	help.BorderLabel = "Help"
	help.BorderFg = ui.ColorCyan

	// build
	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(6, 0, help),
		),
	)

	draw := func(t int) {
		ui.Body.Align()
		ui.Render(ui.Body)
	}

	draw(0)
	ui.Handle("/sys/kbd/q", func(ui.Event) {
		ui.StopLoop()
	})
	ui.Handle("/timer/1s", func(e ui.Event) {
		t := e.Data.(ui.EvtTimer)
		draw(int(t.Count))
	})
	ui.Loop()
}
