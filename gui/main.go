package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"io/fs"
	"os"
)

//go:embed static
var staticFS embed.FS
var version = "0.1.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--engine" {
		runEngine()
		return
	}
	app := newApp()
	assets, _ := fs.Sub(staticFS, "static")
	var appCtx context.Context
	var tray *systemTray
	width, height := 760, 620
	if len(app.config.Profiles) > 1 {
		width, height = 800, 620
	}
	quitting := false

	err := wails.Run(&options.App{
		Title:             "CampusWay",
		Width:             width,
		Height:            height,
		MinWidth:          390,
		MinHeight:         560,
		HideWindowOnClose: true,
		AssetServer:       &assetserver.Options{Assets: assets},
		Bind:              []interface{}{app},
		Windows:           &windows.Options{Theme: windows.SystemDefault},
		OnStartup: func(ctx context.Context) {
			appCtx = ctx
			tray = setupTray(ctx, app, func() {
				quitting = true
				_ = app.Disconnect()
				if tray != nil {
					tray.destroy()
				}
				runtime.Quit(ctx)
			})
			app.resize = func(manager bool) {
				if manager {
					runtime.WindowSetMinSize(ctx, 390, 560)
					runtime.WindowSetSize(ctx, 800, 620)
				} else {
					runtime.WindowSetMinSize(ctx, 390, 560)
					runtime.WindowSetSize(ctx, 460, 600)
				}
			}
		},
		OnBeforeClose: func(ctx context.Context) bool {
			if quitting {
				if err := app.Disconnect(); err != nil {
					_, _ = runtime.MessageDialog(ctx, runtime.MessageDialogOptions{Type: runtime.WarningDialog, Title: "CampusWay", Message: err.Error()})
					return true
				}
				return false
			}
			// Minimize / Hide to tray on close
			runtime.WindowHide(ctx)
			return true
		},
		OnShutdown: func(ctx context.Context) {
			if tray != nil {
				tray.destroy()
			}
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "campusway-desktop-v1",
			OnSecondInstanceLaunch: func(options.SecondInstanceData) {
				if appCtx != nil {
					runtime.WindowUnminimise(appCtx)
					runtime.WindowShow(appCtx)
				}
			},
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
