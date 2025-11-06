//go:build linux

package main

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hajimehoshi/oto/v2"
	"github.com/robfig/cron/v3"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/fish"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/piper"
	"github.com/youpy/go-wav"
)

var otoCtx *oto.Context

func say(piperClient *piper.PiperClient, text string) {
	log.Printf("saying '%s'...", text)
	wavData, err := piperClient.Synthesize(text)
	if err != nil {
		log.Printf("failed to synthesize text: %v", err)
		return
	}

	wavReader := wav.NewReader(bytes.NewReader(wavData))
	format, err := wavReader.Format()
	if err != nil {
		log.Printf("failed to read wav format: %v", err)
		return
	}

	if otoCtx == nil {
		otoCtx, _, err = oto.NewContext(int(format.SampleRate), int(format.NumChannels), oto.FormatSignedInt16LE)
		if err != nil {
			log.Printf("failed to create oto context: %v", err)
			return
		}
	}

	player := otoCtx.NewPlayer(wavReader)
	player.Play()
	for player.IsPlaying() {
		time.Sleep(100 * time.Millisecond)
	}
	if err := player.Close(); err != nil {
		log.Printf("failed to close player: %v", err)
	}
	log.Printf("finished saying '%s'.", text)
}

func main() {
	myFish, err := fish.NewFish("gpiochip0")
	if err != nil {
		log.Fatalf("failed to initialize fish: %v", err)
	}
	defer myFish.Close()

	piperClient := piper.NewPiperClient("http://piper:5000/synthesize")

	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		log.Fatalf("Error loading location: %v", err)
	}

	c := cron.New(cron.WithLocation(loc))

	c.AddFunc("* * * * *", func() {
		myFish.Lock()
		defer myFish.Unlock()

		fmt.Println("Mittag...")
		fmt.Println("Raising body...")
		if err := myFish.RaiseBody(); err != nil {
			log.Printf("Error raising body: %v", err)
		}
		time.Sleep(1 * time.Second)
		fmt.Println("Opening mouth...")
		if err := myFish.OpenMouth(); err != nil {
			log.Printf("Error opening mouth: %v", err)
		}
		say(piperClient, "Mittag")
		time.Sleep(2 * time.Second)
		fmt.Println("Closing mouth...")
		if err := myFish.StopMouth(); err != nil {
			log.Printf("Error closing mouth: %v", err)
		}
		time.Sleep(1 * time.Second)
		fmt.Println("Stopping body...")
		if err := myFish.StopBody(); err != nil {
			log.Printf("Error stopping body: %v", err)
		}
		time.Sleep(1 * time.Second)
		fmt.Println("Tail...")
		if err := myFish.RaiseTail(); err != nil {
			log.Printf("Error stopping body: %v", err)
		}
		time.Sleep(1 * time.Second)
		fmt.Println("Tail...")
		if err := myFish.StopBody(); err != nil {
			log.Printf("Error stopping body: %v", err)
		}
	})
	go c.Start()

	// 5. Keep the main program alive
	select {}

}
