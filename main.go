package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"goals/internal/mcp"
	"goals/internal/store"
	"goals/internal/tray"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// MCP stdio mode: `goals.exe mcp [--db path]` — same single binary.
	// NOTE: handled before wails.Run so MCP never touches the GUI single-instance lock.
	if len(os.Args) > 1 && (os.Args[1] == "mcp" || os.Args[1] == "--mcp") {
		fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
		dbPath := fs.String("db", "", "path to goals.db (or set GOALS_DB_PATH)")
		_ = fs.Parse(os.Args[2:])
		os.Exit(mcp.Run(*dbPath))
	}
	if v := os.Getenv("GOALS_MCP"); v == "1" {
		os.Exit(mcp.Run(""))
	}

	dbPath := os.Getenv("GOALS_DB_PATH")

	// Create an instance of the app structure
	app := NewApp(dbPath)

	// System-tray icon (visible in the hidden-icons area while the app runs,
	// including background timer mode).
	tray.OnShow = func() {
		runtime.WindowShow(app.ctx)
		runtime.WindowUnminimise(app.ctx)
	}
	tray.OnQuit = func() {
		app.QuitApp()
	}
	tray.Start()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "Goals",
		Width:     1280,
		Height:    800,
		MinWidth:  940,
		MinHeight: 600,
		// Frameless: no OS chrome at all — the in-app menubar is the
		// title bar (drag region + normal caption buttons).
		// No native top menu bar either — all actions live in-app.
		Frameless: true,
		Menu: menu.NewMenu(),
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 9, G: 9, B: 15, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		// Closing the window keeps the app (and any running timer) alive in
		// the background. Launch goals.exe again to restore the window.
		HideWindowOnClose: true,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "goals-planner-single-instance",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				runtime.WindowShow(app.ctx)
				runtime.WindowUnminimise(app.ctx)
			},
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}

	_ = fmt.Sprintf
	_ = store.ResolveDBPath
}
