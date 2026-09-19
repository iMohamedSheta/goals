// Package tray manages the Windows system-tray icon and its menu.
package tray

import (
	_ "embed"
	"sync"

	"github.com/getlantern/systray"
)

//go:embed icon.ico
var iconICO []byte

// Callbacks wired by main.go.
var (
	OnShow = func() {}
	OnQuit = func() {}
	ready  sync.Once
)

// Start runs the system-tray icon loop on its own OS thread (alongside Wails).
// The icon keeps the app visible in the tray / hidden-icons area while the
// window is closed (background timer mode).
func Start() {
	ready.Do(func() {
		go systray.Run(onReady, onExit)
	})
}

func onReady() {
	systray.SetIcon(iconICO)
	systray.SetTitle("Goals")
	systray.SetTooltip("Goals")

	mShow := systray.AddMenuItem("Show / إظهار", "Restore the Goals window")
	mQuit := systray.AddMenuItem("Quit / خروج", "Stop the timer and quit Goals")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				OnShow()
			case <-mQuit.ClickedCh:
				OnQuit()
			}
		}
	}()
}

func onExit() {}

// SetTooltip updates the hover text, e.g. with the live timer.
func SetTooltip(s string) {
	defer func() { _ = recover() }()
	systray.SetTooltip(s)
}

// Stop removes the tray icon.
func Stop() {
	defer func() { _ = recover() }()
	systray.Quit()
}
