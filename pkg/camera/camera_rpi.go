//go:build linux && arm64

package camera

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

// captureFrameRPi captures a frame using libcamera-apps on Raspberry Pi
// This is optimized for Raspberry Pi Camera Module v3 with its complex ISP path
// Uses libcamera-jpeg which properly handles the Module 3's ISP
func (c *Camera) captureFrameRPi() ([]byte, error) {
	// Use libcamera-jpeg to capture a single frame
	// This is the most reliable method for Camera Module v3
	// The camera's ISP (Image Signal Processor) requires libcamera-apps
	// Standard V4L2 libraries struggle with the Module 3's ISP complexity

	cmd := exec.Command(
		"libcamera-jpeg",
		"--width", fmt.Sprintf("%d", c.width),
		"--height", fmt.Sprintf("%d", c.height),
		"--timeout", "1", // 1ms timeout for quick capture
		"--nopreview",   // No display window
		"--output", "-", // Output to stdout
		"--quality", "80", // JPEG quality (0-100)
		"--encoding", "jpg", // Explicit JPEG encoding
		// Module 3 specific optimizations
		"--awb", "auto", // Auto white balance
		"--metering", "average", // Average metering mode
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// If libcamera-jpeg fails, try to provide helpful error info
		log.Printf("libcamera-jpeg error: %v, stderr: %s", err, stderr.String())
		return nil, fmt.Errorf("libcamera-jpeg failed: %w (ensure libcamera-apps is installed)", err)
	}

	return stdout.Bytes(), nil
}

// checkLibcameraInstalled checks if libcamera-apps are installed
func checkLibcameraInstalled() error {
	cmd := exec.Command("which", "libcamera-jpeg")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("libcamera-apps not found: install with 'sudo apt install -y libcamera-apps'")
	}
	return nil
}

// Alternative: For continuous streaming (more efficient than repeated captures)
// This can be used in future optimizations
func (c *Camera) startLibcameraVidStream() error {
	// libcamera-vid can output MJPEG stream continuously
	// More efficient than repeated libcamera-jpeg calls
	cmd := exec.Command(
		"libcamera-vid",
		"--width", fmt.Sprintf("%d", c.width),
		"--height", fmt.Sprintf("%d", c.height),
		"--timeout", "0", // Run indefinitely
		"--nopreview",
		"--codec", "mjpeg", // MJPEG output
		"--inline",      // Inline headers
		"--listen",      // Listen mode
		"--output", "-", // Output to stdout
		"--framerate", fmt.Sprintf("%d", c.fps),
	)

	// This would need pipe handling to read continuous MJPEG stream
	// Left as future optimization - current frame-by-frame approach works well

	log.Printf("Note: libcamera-vid streaming available for future optimization")
	return cmd.Start()
}
