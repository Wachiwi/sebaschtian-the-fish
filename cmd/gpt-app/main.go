package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/warthog618/go-gpiocdev"
	"github.com/warthog618/go-gpiocdev/device/rpi"
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
	c, err := gpiocdev.NewChip("gpiochip0")
	if err != nil {
	}

	defer c.Close()

	// enableBodyPin, _ := c.RequestLine(rpi.GPIO12, gpiocdev.AsOutput())
	// in3Pin, _ := c.RequestLine(rpi.GPIO26, gpiocdev.AsOutput())
	// in4Pin, _ := c.RequestLine(rpi.GPIO19, gpiocdev.AsOutput())

	enableHeadPin, _ := c.RequestLine(rpi.GPIO5, gpiocdev.AsOutput())
	in1Pin, _ := c.RequestLine(rpi.GPIO13, gpiocdev.AsOutput())
	in2Pin, _ := c.RequestLine(rpi.GPIO6, gpiocdev.AsOutput())

	for {
		err := process(ctx, enableHeadPin, in1Pin, in2Pin)
		if err != nil {
			fmt.Printf("Error processing: %v", err)
		}
	}
}
func process(ctx context.Context, head *gpiocdev.Line, in1 *gpiocdev.Line, in2 *gpiocdev.Line) error {
	log.Printf("Open")

	_ = head.SetValue(1)
	_ = in1.SetValue(1)
	_ = in2.SetValue(0)
	return nil
}
