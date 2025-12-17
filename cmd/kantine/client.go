package main

import (
	"encoding/json"
	"fmt"

	"github.com/wachiwi/sebaschtian-the-fish/pkg/kantine"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/logger"
)

func main() {
	logger.Setup()
	menu, err := kantine.Fetch()
	if err != nil {
		logger.Fatal("Failed to fetch menu", "error", err)
	}
	jsonData, err := json.MarshalIndent(menu, "", "  ")
	if err != nil {
		logger.Fatal("Error marshalling to JSON", "error", err)
	}
	fmt.Println(string(jsonData))
}
