//go:build linux

package fish

import (
	"fmt"
	"sync"

	"github.com/warthog618/go-gpiocdev"
	"github.com/warthog618/go-gpiocdev/device/rpi"
)

// Motor represents a single DC motor controlled by an H-Bridge.
type Motor struct {
	enable *gpiocdev.Line
	in1    *gpiocdev.Line
	in2    *gpiocdev.Line
}

// Forward turns the motor in the forward direction.
func (m *Motor) Forward() error {
	if err := m.in1.SetValue(1); err != nil {
		return err
	}
	if err := m.in2.SetValue(0); err != nil {
		return err
	}
	return m.enable.SetValue(1)
}

// Reverse turns the motor in the reverse direction.
func (m *Motor) Reverse() error {
	if err := m.in1.SetValue(0); err != nil {
		return err
	}
	if err := m.in2.SetValue(1); err != nil {
		return err
	}
	return m.enable.SetValue(1)
}

// Stop halts the motor.
func (m *Motor) Stop() error {
	if err := m.in1.SetValue(0); err != nil {
		return err
	}
	if err := m.in2.SetValue(0); err != nil {
		return err
	}
	return m.enable.SetValue(0)
}

// Fish represents the fish with its controllable parts.
type Fish struct {
	mu        sync.Mutex
	chip      *gpiocdev.Chip
	HeadMotor *Motor
	BodyMotor *Motor
}

// NewFish initializes the GPIO pins and returns a new Fish object.
func NewFish(chipName string) (*Fish, error) {
	c, err := gpiocdev.NewChip(chipName)
	if err != nil {
		return nil, fmt.Errorf("failed to open chip: %w", err)
	}

	// Head motor pins
	enableHeadPin, err := c.RequestLine(rpi.GPIO5, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, err
	}
	in1Pin, err := c.RequestLine(rpi.GPIO13, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, err
	}
	in2Pin, err := c.RequestLine(rpi.GPIO6, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, err
	}

	// Body motor pins
	enableBodyPin, err := c.RequestLine(rpi.GPIO12, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, err
	}
	in3Pin, err := c.RequestLine(rpi.GPIO26, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, err
	}
	in4Pin, err := c.RequestLine(rpi.GPIO19, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, err
	}

	fish := &Fish{
		chip: c,
		HeadMotor: &Motor{
			enable: enableHeadPin,
			in1:    in1Pin,
			in2:    in2Pin,
		},
		BodyMotor: &Motor{
			enable: enableBodyPin,
			in1:    in3Pin,
			in2:    in4Pin,
		},
	}

	return fish, nil
}

func (f *Fish) Lock() {
	f.mu.Lock()
}

func (f *Fish) Unlock() {
	f.mu.Unlock()
}

// Close releases all GPIO resources.
func (f *Fish) Close() {
	f.HeadMotor.Stop()
	f.BodyMotor.Stop()
	f.chip.Close()
}

// OpenMouth moves the head motor to open the mouth.
func (f *Fish) OpenMouth() error {
	return f.HeadMotor.Forward()
}

// CloseMouth moves the head motor to close the mouth.
func (f *Fish) CloseMouth() error {
	return f.HeadMotor.Reverse()
}

// StopMouth stops the head motor.
func (f *Fish) StopMouth() error {
	return f.HeadMotor.Stop()
}

// RaiseBody moves the body motor to raise the body.
func (f *Fish) RaiseBody() error {
	return f.BodyMotor.Forward()
}

// RaiseTail moves the body motor to raise the tail.
func (f *Fish) RaiseTail() error {
	return f.BodyMotor.Reverse()
}

// StopBody stops the body motor.
func (f *Fish) StopBody() error {
	return f.BodyMotor.Stop()
}

