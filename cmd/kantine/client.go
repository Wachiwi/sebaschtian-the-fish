package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/wachiwi/sebaschtian-the-fish/pkg/kantine"
)

func main() {
	menu := kantine.Fetch()
	jsonData, err := json.MarshalIndent(menu, "", "  ")
	if err != nil {
		log.Fatalf("Error marshalling to JSON: %v", err)
	}
	fmt.Println(string(jsonData))
}
