package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	rpio "github.com/warthog618/gpio"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:           "app",
	Short:         "Main application to handle speech-to-text, chat-gpt and text-to-speech",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		run(ctx)
		return nil
	},
}

func run(ctx context.Context) {
	for {
		err := process(ctx)
		if err != nil {
			fmt.Printf("Error processing: %v", err)
		}
	}
}

func process(ctx context.Context) error {
	err := rpio.Open()
	defer rpio.Close()

	if err != nil {
		return err
	}
	log.Printf("Open")

	enableHeadPin := rpio.NewPin(5)
	in3Pin := rpio.NewPin(6)
	in4Pin := rpio.NewPin(13)

	enableHeadPin.Output()
	in3Pin.Output()
	in4Pin.Output()

	// Enable the motor
	enableHeadPin.High()
	in3Pin.Low()
	in4Pin.High()
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)
	log.Printf("Close")

	enableHeadPin.Output()
	in3Pin.Output()
	in4Pin.Output()

	enableHeadPin.Low()
	in3Pin.Low()
	in4Pin.Low()

	if err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return nil
}
