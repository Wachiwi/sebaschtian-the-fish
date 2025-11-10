//go:build linux

package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/oto/v2"
	"github.com/robfig/cron/v3"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/fish"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/piper"
	"github.com/youpy/go-wav"
)

var otoCtx *oto.Context

type Phrase struct {
	Text   string
	Weight int
}

func getWeightedRandomPhrase() string {
	now := time.Now()
	hour := now.Hour()
	day := now.Day()

	phrases := []Phrase{
		{Text: "Bald ist Mittag", Weight: 10},     // Base weight for lunch
		{Text: "Bald ist Feierabend", Weight: 10}, // Base weight for end of work
		{Text: "Es ist spät, Zeit für Magic!", Weight: 10},
		{Text: "Feierabend, wie das duftet. Kräftig, deftig, würzig gut!", Weight: 10}, // Base weight for end of work
		{Text: "Es ist Mittwoch, meine Kerle.", Weight: 50},
		{Text: "Komm in die Gruppe! Hinterbüro ist beste!", Weight: 50},
		{Text: "Hallo, I bims. Vong Fisch Sprache her.", Weight: 50},
		{Text: "Der Gerät wird nie müde. Der Gerät schläft nie ein. Der Gerät ist immer vor die Chef im Geschäft.", Weight: 50},
		{Text: "Haben wir noch Peps da?", Weight: 50},
		{Text: "Läuft bei uns. Ich mach nix, bin aber auch nicht billable.", Weight: 50},
	}

	// Adjust weights based on the time
	if hour >= 11 && hour <= 12 {
		phrases[0].Weight += 70
	}
	if hour >= 15 && hour <= 17 {
		phrases[1].Weight += 70
		phrases[2].Weight += 70
	}
	if hour >= 17 && hour <= 19 {
		phrases[3].Weight += 70
	}

	if day == 3 {
		phrases[4].Weight += 100
	} else {
		phrases[4].Weight = 0
	}

	// Calculate total weight
	totalWeight := 0
	for _, p := range phrases {
		totalWeight += p.Weight
	}

	// Generate a random number and select a phrase
	r := rand.Intn(totalWeight)
	for _, p := range phrases {
		r -= p.Weight
		if r < 0 {
			return p.Text
		}
	}

	return phrases[0].Text // Fallback to the default phrase
}

func say(piperClient *piper.PiperClient, text string) {
	log.Printf("saying '%s'...", text)
	wavData, err := piperClient.Synthesize(text)
	if err != nil {
		log.Printf("failed to synthesize text: %v", err)
		return
	}

	wavReader := wav.NewReader(bytes.NewReader(wavData))
	if otoCtx == nil {
		log.Printf("audio context not initialized")
		return
	}

	player := otoCtx.NewPlayer(wavReader)
	player.Play()
	time.Sleep(1 * time.Second)
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

	// Initialize audio context at startup with standard CD quality settings
	// This ensures the audio device is active and ready before first use
	log.Println("Initializing audio context...")
	var ready chan struct{}
	otoCtx, ready, err = oto.NewContext(22050, 1, oto.FormatSignedInt16LE)
	if err != nil {
		log.Fatalf("failed to create oto context: %v", err)
	}
	<-ready
	log.Println("Audio context ready")

	piperClient := piper.NewPiperClient("http://piper:5000")

	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		log.Fatalf("Error loading location: %v", err)
	}

	c := cron.New(cron.WithLocation(loc))

	c.AddFunc("* * * * *", func() {
		myFish.Lock()
		defer myFish.Unlock()

		phraseToSay := getWeightedRandomPhrase()
		fmt.Printf("%s...\n", phraseToSay)
		fmt.Println("Raising body...")
		if err := myFish.RaiseBody(); err != nil {
			log.Printf("Error raising body: %v", err)
		}
		time.Sleep(1 * time.Second)
		fmt.Println("Opening mouth...")
		if err := myFish.OpenMouth(); err != nil {
			log.Printf("Error opening mouth: %v", err)
		}
		say(piperClient, phraseToSay)
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
