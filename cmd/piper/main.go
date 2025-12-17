package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/wachiwi/sebaschtian-the-fish/pkg/logger"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/piper"
)

func main() {
	logger.Setup()
	var text string
	var outputFile string

	flag.StringVar(&text, "text", "Hallo Yebba", "Text to synthesize")
	flag.StringVar(&outputFile, "output", "test.wav", "Output file path")
	flag.Parse()

	if text == "" {
		logger.Fatal("Text to synthesize cannot be empty")
	}

	client := piper.NewPiperClient("http://localhost:10200")
	audioData, err := client.Synthesize(text)
	if err != nil {
		logger.Fatal("Failed to synthesize text", "error", err)
	}

	err = os.WriteFile(outputFile, audioData, 0644)
	if err != nil {
		logger.Fatal("Failed to write audio to file", "error", err)
	}

	slog.Info("Successfully synthesized text", "file", outputFile)
}
