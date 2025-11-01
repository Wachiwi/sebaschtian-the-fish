package main

import (
	"fmt"
	"log"
	"time"

	"github.com/wachiwi/sebaschtian-the-fish/pkg/fish"
)

func main() {
	myFish, err := fish.NewFish("gpiochip0")
	if err != nil {
		log.Fatalf("failed to initialize fish: %v", err)
	}
	defer myFish.Close()

	// Example sequence to demonstrate the new API.
	// This can be replaced with the actual application logic.
	for {
		// fmt.Println("Opening mouth...")
		// if err := myFish.OpenMouth(); err != nil {
		// 	log.Printf("Error opening mouth: %v", err)
		// }
		// time.Sleep(2 * time.Second)

		// fmt.Println("Closing mouth...")
		// if err := myFish.CloseMouth(); err != nil {
		// 	log.Printf("Error closing mouth: %v", err)
		// }
		// time.Sleep(2 * time.Second)

		fmt.Println("Raising body...")
		if err := myFish.RaiseBody(); err != nil {
			log.Printf("Error raising body: %v", err)
		}
		time.Sleep(2 * time.Second)

		fmt.Println("Stop body...")
		if err := myFish.StopBody(); err != nil {
			log.Printf("Error raising tail: %v", err)
		}

		// fmt.Println("Raising tail...")
		// if err := myFish.RaiseTail(); err != nil {
		// 	log.Printf("Error raising tail: %v", err)
		// }
		// time.Sleep(2 * time.Second)

		// myFish.StopBody()
		// myFish.StopMouth()
		time.Sleep(1 * time.Second)
	}
}
