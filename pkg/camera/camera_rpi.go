//go:build linux && arm64

package camera

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

// captureFrameRPi captures a frame using libcamera-apps/rpicam-apps on Raspberry Pi
// This is optimized for Raspberry Pi Camera Module v3 with its complex ISP path
// Tries rpicam-jpeg first (new name), falls back to libcamera-jpeg (old name)
func (c *Camera) captureFrameRPi() ([]byte, error) {
	// Newer Raspberry Pi OS (Bookworm+) uses rpicam-* commands
	// Older versions use libcamera-* commands
	// Try both for maximum compatibility

	// Try rpicam-jpeg first (newer)
	frame, err := c.captureWithCommand("rpicam-jpeg")
	if err == nil {
		return frame, nil
	}

	// Fall back to libcamera-jpeg (older)
	frame, err = c.captureWithCommand("libcamera-jpeg")
	if err == nil {
		return frame, nil
	}

	// Neither worked - provide helpful error
	return nil, fmt.Errorf("camera capture failed: neither rpicam-jpeg nor libcamera-jpeg found. Install with: sudo apt install -y rpicam-apps (or libcamera-apps for older OS)")
}

// captureWithCommand captures a frame using the specified command
func (c *Camera) captureWithCommand(cmdName string) ([]byte, error) {
	// Use rpicam-jpeg/libcamera-jpeg to capture a single frame
	// This is the most reliable method for Camera Module v3
	// The camera's ISP (Image Signal Processor) requires these tools
	// Standard V4L2 libraries struggle with the Module 3's ISP complexity

	cmd := exec.Command(
		cmdName,
		"--width", fmt.Sprintf("%d", c.width),
		"--height", fmt.Sprintf("%d", c.height),
		"--timeout", "1", // 1ms timeout for quick capture
		"--nopreview",   // No display window
		"--output", "-", // Output to stdout
		"--quality", "80", // JPEG quality (0-100)
		// Module 3 specific optimizations
		"--awb", "auto", // Auto white balance
		"--metering", "average", // Average metering mode
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s failed: %w (stderr: %s)", cmdName, err, stderr.String())
	}

	if stdout.Len() == 0 {
		return nil, fmt.Errorf("%s returned empty frame", cmdName)
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
