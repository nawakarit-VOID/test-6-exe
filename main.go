// Copyright (c) 2026 Nawakarit
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License v3.0.
package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// ─── State ────────────────────────────────────────────────────────────────────

type Mode int

const (
	ModeWork Mode = iota
	ModeShortBreak
	ModeLongBreak
)

var modeNames = map[Mode]string{
	ModeWork:       "🍅  Work",
	ModeShortBreak: "☕  Short Break",
	ModeLongBreak:  "🌙  Long Break",
}

var modeDurations = map[Mode]int{
	ModeWork:       25 * 60,
	ModeShortBreak: 5 * 60,
	ModeLongBreak:  15 * 60,
}

var modeColors = map[Mode]color.Color{
	ModeWork:       color.RGBA{R: 220, G: 80, B: 80, A: 255},
	ModeShortBreak: color.RGBA{R: 70, G: 170, B: 140, A: 255},
	ModeLongBreak:  color.RGBA{R: 70, G: 120, B: 200, A: 255},
}

// ─── App ──────────────────────────────────────────────────────────────────────

type PomodoroApp struct {
	fyneApp   fyne.App
	window    fyne.Window
	mode      Mode
	remaining int
	running   bool
	sessions  int
	ticker    *time.Ticker
	stopChan  chan struct{}

	// UI elements
	timerLabel   *canvas.Text
	modeLabel    *canvas.Text
	sessionLabel *canvas.Text
	startBtn     *widget.Button
	resetBtn     *widget.Button
	bgRect       *canvas.Rectangle
	progressBar  *widget.ProgressBar
}

func newPomodoroApp() *PomodoroApp {
	p := &PomodoroApp{
		mode:      ModeWork,
		remaining: modeDurations[ModeWork],
		stopChan:  make(chan struct{}),
	}
	return p
}

func (p *PomodoroApp) formatTime() string {
	m := p.remaining / 60
	s := p.remaining % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func (p *PomodoroApp) progress() float64 {
	total := modeDurations[p.mode]
	if total == 0 {
		return 0
	}
	return float64(total-p.remaining) / float64(total)
}

func (p *PomodoroApp) updateUI() {
	p.timerLabel.Text = p.formatTime()
	p.timerLabel.Refresh()

	p.modeLabel.Text = modeNames[p.mode]
	p.modeLabel.Refresh()

	p.sessionLabel.Text = fmt.Sprintf("Sessions completed: %d", p.sessions)
	p.sessionLabel.Refresh()

	p.bgRect.FillColor = modeColors[p.mode]
	p.bgRect.Refresh()

	p.progressBar.SetValue(p.progress())

	if p.running {
		p.startBtn.SetText("⏸  Pause")
	} else {
		p.startBtn.SetText("▶  Start")
	}
}

func (p *PomodoroApp) startTicker() {
	p.stopChan = make(chan struct{})
	p.ticker = time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-p.ticker.C:
				if p.remaining > 0 {
					p.remaining--
					p.updateUI()
				} else {
					p.ticker.Stop()
					p.running = false
					p.onTimerDone()
				}
			case <-p.stopChan:
				return
			}
		}
	}()
}

func (p *PomodoroApp) onTimerDone() {
	if p.mode == ModeWork {
		p.sessions++
		if p.sessions%4 == 0 {
			p.mode = ModeLongBreak
		} else {
			p.mode = ModeShortBreak
		}
	} else {
		p.mode = ModeWork
	}
	p.remaining = modeDurations[p.mode]
	p.updateUI()

	// Simple notification via window title
	p.window.SetTitle("🔔 Timer Done! - Pomodoro")
	go func() {
		time.Sleep(3 * time.Second)
		p.window.SetTitle("🍅 Pomodoro Timer")
	}()
}

func (p *PomodoroApp) toggleStartPause() {
	if p.running {
		// Pause
		p.running = false
		close(p.stopChan)
		if p.ticker != nil {
			p.ticker.Stop()
		}
	} else {
		// Start
		p.running = true
		p.startTicker()
	}
	p.updateUI()
}

func (p *PomodoroApp) reset() {
	p.running = false
	if p.ticker != nil {
		p.ticker.Stop()
	}
	select {
	case p.stopChan <- struct{}{}:
	default:
	}
	p.remaining = modeDurations[p.mode]
	p.updateUI()
}

func (p *PomodoroApp) switchMode(mode Mode) {
	p.running = false
	if p.ticker != nil {
		p.ticker.Stop()
	}
	select {
	case p.stopChan <- struct{}{}:
	default:
	}
	p.mode = mode
	p.remaining = modeDurations[mode]
	p.updateUI()
}

func (p *PomodoroApp) buildUI() {
	// Background
	p.bgRect = canvas.NewRectangle(modeColors[ModeWork])

	// Mode label (top)
	p.modeLabel = canvas.NewText(modeNames[ModeWork], color.White)
	p.modeLabel.TextSize = 18
	p.modeLabel.Alignment = fyne.TextAlignCenter
	p.modeLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Timer label (big)
	p.timerLabel = canvas.NewText(p.formatTime(), color.White)
	p.timerLabel.TextSize = 72
	p.timerLabel.Alignment = fyne.TextAlignCenter
	p.timerLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

	// Progress bar
	p.progressBar = widget.NewProgressBar()
	p.progressBar.SetValue(0)

	// Session count
	p.sessionLabel = canvas.NewText(fmt.Sprintf("Sessions completed: %d", p.sessions), color.White)
	p.sessionLabel.TextSize = 13
	p.sessionLabel.Alignment = fyne.TextAlignCenter

	// Buttons
	p.startBtn = widget.NewButton("▶  Start", func() {
		p.toggleStartPause()
	})
	p.resetBtn = widget.NewButton("↺  Reset", func() {
		p.reset()
	})

	// Mode switcher buttons
	workBtn := widget.NewButton("🍅 Work", func() { p.switchMode(ModeWork) })
	shortBtn := widget.NewButton("☕ Short", func() { p.switchMode(ModeShortBreak) })
	longBtn := widget.NewButton("🌙 Long", func() { p.switchMode(ModeLongBreak) })
	modeRow := container.New(layout.NewGridLayout(3), workBtn, shortBtn, longBtn)

	// Layout
	controlRow := container.New(layout.NewGridLayout(2), p.startBtn, p.resetBtn)

	timerBox := container.NewVBox(
		layout.NewSpacer(),
		p.modeLabel,
		p.timerLabel,
		p.progressBar,
		p.sessionLabel,
		layout.NewSpacer(),
		controlRow,
		modeRow,
		layout.NewSpacer(),
	)

	content := container.NewMax(p.bgRect, container.NewPadded(timerBox))
	p.window.SetContent(content)
}

func (p *PomodoroApp) run() {
	p.fyneApp = app.NewWithID("com.nawakarit.pomodoro")
	icon := fyne.NewStaticResource("icon.png", iconData)
	p.fyneApp.SetIcon(icon)
	p.window = p.fyneApp.NewWindow("🍅 Pomodoro Timer")
	p.window.SetIcon(icon)
	p.window.Resize(fyne.NewSize(360, 420))
	p.window.SetFixedSize(true)

	p.buildUI()
	p.window.ShowAndRun()
}

//go:embed icon.png
var iconData []byte

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	a := app.NewWithID("com.nawakarit.pomodoro")
	icon := fyne.NewStaticResource("icon.png", iconData)
	a.SetIcon(icon)
	w := a.NewWindow("Pomodoro")
	w.SetIcon(icon)

	app := newPomodoroApp()
	app.run()
}
