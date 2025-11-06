//go:build linux

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/fish"
)

func main() {
	myFish, err := fish.NewFish("gpiochip0")
	if err != nil {
		log.Fatalf("failed to initialize fish: %v", err)
	}
	defer myFish.Close()

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

