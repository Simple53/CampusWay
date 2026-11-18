//go:build !tun

package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/containers/winquit/pkg/winquit"
	"github.com/nnu-connect/nnu-connect/configs"
	"github.com/nnu-connect/nnu-connect/engine"
	"github.com/nnu-connect/nnu-connect/log"
)

var conf configs.Config

func main() {
	log.Init()

	if CommitID != "" {
		log.Println("Start RouteDesk CLI v" + routeDeskVersion + "-" + CommitID)
	} else {
		log.Println("Start RouteDesk CLI v" + routeDeskVersion)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if runtime.GOOS == "windows" {
			done := make(chan os.Signal, 1)
			signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
			winquit.SimulateSigTermOnQuit(done)
			<-done
		} else {
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
			<-quit
		}
		log.Println("Received termination signal, stopping...")
		cancel()
	}()

	err := engine.Run(ctx, conf, func(status string, msg string) {
		log.Printf("[%s] %s", status, msg)
	})

	if err != nil {
		log.Fatalf("RouteDesk execution failed: %v", err)
	}
}
