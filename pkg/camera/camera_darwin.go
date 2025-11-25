//go:build darwin

package camera

import (
	"bytes"
	"fmt"
	"os/exec"
)

// captureFrameRPi captures from the default macOS webcam using ffmpeg
// This allows local development with actual camera input
func (c *Camera) captureFrameRPi() ([]byte, error) {
	return c.captureFrameMacWebcam()
}

// captureFrameMacWebcam captures a frame from the default macOS webcam using ffmpeg
// Requires ffmpeg to be installed: brew install ffmpeg
func (c *Camera) captureFrameMacWebcam() ([]byte, error) {
	// Use ffmpeg to capture a single frame from the default camera (0:0 = FaceTime camera)
	// -f avfoundation: Use AVFoundation framework (macOS camera API)
	// -i "0": Default video device (usually FaceTime camera)
	// -frames:v 1: Capture only 1 frame
	// -f image2pipe: Output as image stream
	// -vcodec mjpeg: Encode as JPEG
	// pipe:1: Output to stdout

	cmd := exec.Command(
		"ffmpeg",
		"-f", "avfoundation",
		"-video_size", fmt.Sprintf("%dx%d", c.width, c.height),
		"-framerate", fmt.Sprintf("%d", c.fps),
		"-i", "0", // Device 0 = default camera
		"-frames:v", "1", // Capture single frame
		"-f", "image2pipe", // Output as image
		"-vcodec", "mjpeg", // JPEG encoding
		"-q:v", "5", // Quality (2-31, lower is better)
		"-hide_banner",       // Hide ffmpeg banner
		"-loglevel", "error", // Only show errors
		"pipe:1", // Output to stdout
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg capture failed: %w, stderr: %s (install with: brew install ffmpeg)", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// listMacCameras lists available camera devices on macOS
func listMacCameras() (string, error) {
	cmd := exec.Command("ffmpeg", "-f", "avfoundation", "-list_devices", "true", "-i", "")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// This command intentionally fails, but outputs device list to stderr
	cmd.Run()

	return stderr.String(), nil
}
