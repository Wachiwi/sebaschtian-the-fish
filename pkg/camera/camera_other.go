//go:build !darwin && !(linux && arm64)

package camera

import "fmt"

// captureFrameRPi is a stub for non-RPi platforms
func (c *Camera) captureFrameRPi() ([]byte, error) {
	return nil, fmt.Errorf("raspberry pi camera not available on this platform")
}
