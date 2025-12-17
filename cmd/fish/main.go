//go:build linux

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/fish"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/logger"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/piper"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/telemetry"
)

func main() {
	logger.Setup()

	// Initialize Telemetry
	ctx := context.Background()
	shutdown, err := telemetry.Setup(ctx, "sebaschtian-fish")
	if err != nil {
		slog.Error("Failed to setup telemetry", "error", err)
	} else {
		defer func() {
			if err := shutdown(context.Background()); err != nil {
				slog.Error("Error shutting down telemetry", "error", err)
			}
		}()
	}

	myFish, err := fish.NewFish("gpiochip0")
	if err != nil {
		logger.Fatal("failed to initialize fish", "error", err)
	}
	defer myFish.Close()

	slog.Info("Audio system ready.")

	piperClient := piper.NewPiperClient("http://piper:5000")

	myFish.Say(piperClient, "Hallo Ich bins! Bin wieder da und ready!")
	myFish.Lock()
	myFish.StopBody()
	myFish.StopMouth()
	myFish.Unlock()

	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		logger.Fatal("Error loading location", "error", err)
	}

	c := cron.New(
		cron.WithLocation(loc),
		cron.WithChain(cron.SkipIfStillRunning(&logger.CronLogger{Logger: slog.Default()})),
	)

	soundDir := "/sound-data"
	enableTTS := true

	c.AddFunc("* * * * *", func() {
		runFishCycle(myFish, piperClient, soundDir, enableTTS)
	})
	go c.Start()

	select {}
}
