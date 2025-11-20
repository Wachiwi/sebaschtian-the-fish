//go:build linux

package main

import (
	"log"
	"os"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/fish"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/piper"
)

func main() {
	log.SetOutput(os.Stdout)

	myFish, err := fish.NewFish("gpiochip0")
	if err != nil {
		log.Fatalf("failed to initialize fish: %v", err)
	}
	defer myFish.Close()

	log.Println("Audio system ready.")

	piperClient := piper.NewPiperClient("http://piper:5000")

	myFish.Say(piperClient, "Hallo Ich bins! Bin wieder da und ready!")
	myFish.Lock()
	myFish.StopBody()
	myFish.StopMouth()
	myFish.Unlock()

	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		log.Fatalf("Error loading location: %v", err)
	}

	c := cron.New(
		cron.WithLocation(loc),
		cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)),
	)

	soundDir := "/sound-data"
	enableTTS := true

	c.AddFunc("* * * * *", func() {
		runFishCycle(myFish, piperClient, soundDir, enableTTS)
	})
	go c.Start()

	select {}
}
