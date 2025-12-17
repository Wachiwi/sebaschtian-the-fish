//go:build darwin

package main

import (
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/fish"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/logger"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/piper"
)

func main() {
	logger.Setup()

	myFish, err := fish.NewFish("") // Empty string for chipName on macOS
	if err != nil {
		logger.Fatal("failed to initialize fish", "error", err)
	}
	defer myFish.Close()

	slog.Info("Audio system ready (macOS/Darwin mode - no GPIO).")

	// For macOS testing, you can either:
	// 1. Comment out the piper client and Say() call (default)
	// 2. Run a local piper server
	// 3. Change the URL to a remote piper server
	piperClient := piper.NewPiperClient("http://localhost:10200")

	// Test audio without TTS
	slog.Info("Starting fish in macOS mode...")
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

	soundDir := "./sound-data"
	enableTTS := true // Set to true if you have piper running

	c.AddFunc("* * * * *", func() {
		runFishCycle(myFish, piperClient, soundDir, enableTTS)
	})
	go c.Start()

	select {}
}
