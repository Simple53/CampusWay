//go:build !windows

package main

import "context"

type systemTray struct{}

func setupTray(ctx context.Context, app *AppServer, onExit func()) *systemTray {
	return &systemTray{}
}

func (t *systemTray) destroy() {}
