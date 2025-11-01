package main

import (
	"log"

	"github.com/warthog618/go-gpiocdev"
	"github.com/warthog618/go-gpiocdev/device/rpi"
)

func main() {
	c, err := gpiocdev.NewChip("gpiochip0")
	if err != nil {
	}

	defer c.Close()

	enableBodyPin, err := c.RequestLine(rpi.GPIO12, gpiocdev.AsOutput(0))
	if err != nil {
		log.Fatal(err)
	}
	defer enableBodyPin.Close()

	in3Pin, err := c.RequestLine(rpi.GPIO26, gpiocdev.AsOutput(0))
	if err != nil {
		log.Fatal(err)
	}
	defer in3Pin.Close()

	in4Pin, err := c.RequestLine(rpi.GPIO19, gpiocdev.AsOutput(0))
	if err != nil {
		log.Fatal(err)
	}
	defer in4Pin.Close()

	enableHeadPin, _ := c.RequestLine(rpi.GPIO5, gpiocdev.AsOutput(0, 1))
	in1Pin, _ := c.RequestLine(rpi.GPIO13, gpiocdev.AsOutput(0))
	in2Pin, _ := c.RequestLine(rpi.GPIO6, gpiocdev.AsOutput())

	for {
		err = enableBodyPin.SetValue(1)
		if err != nil {
			log.Fatal(err)
		}
		err = in3Pin.SetValue(0)
		if err != nil {
			log.Fatal(err)
		}
		err = in4Pin.SetValue(1)
		if err != nil {
			log.Fatal(err)
		}
	}
}
