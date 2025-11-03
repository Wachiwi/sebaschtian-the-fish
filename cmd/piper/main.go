package main

import (
	"flag"
	"log"
	"os"

	"github.com/wachiwi/sebaschtian-the-fish/pkg/piper"
)

func main() {
	var text string
	var outputFile string

	flag.StringVar(&text, "text", "Hallo Yebba", "Text to synthesize")
	flag.StringVar(&outputFile, "output", "test.wav", "Output file path")
	flag.Parse()

	if text == "" {
		log.Fatal("Text to synthesize cannot be empty")
	}

	client := piper.NewPiperClient("http://localhost:10200")
	audioData, err := client.Synthesize(text)
	if err != nil {
		log.Fatalf("Failed to synthesize text: %v", err)
	}

	err = os.WriteFile(outputFile, audioData, 0644)
	if err != nil {
		log.Fatalf("Failed to write audio to file: %v", err)
	}

	log.Printf("Successfully synthesized text to %s\n", outputFile)
}
