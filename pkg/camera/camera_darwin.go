//go:build darwin

package camera

import (
	"fmt"
	"log/slog"
	"os/exec"
)

var streamingCmd *exec.Cmd

// captureFrameRPi captures from the default macOS webcam using ffmpeg
// This allows local development with actual camera input
func (c *Camera) captureFrameRPi() ([]byte, error) {
	CmdMutex.Lock()
	defer CmdMutex.Unlock()

	// 1. Initialize streaming if not already running
	if streamingCmd == nil {
		if err := c.startStreamingProcess(); err != nil {
			return nil, err
		}
	}

	// 2. Return the most recent frame captured by the background pump
	return GetCurrentFrame()
}

// startStreamingProcess starts the ffmpeg process in MJPEG streaming mode.
func (c *Camera) startStreamingProcess() error {
	// Use ffmpeg to capture frames from the default camera and stream MJPEG to stdout
	// -f avfoundation: Use AVFoundation framework (macOS camera API)
	// -framerate MUST be 30 for most Mac cameras (they don't support arbitrary framerates)
	// -video_size: Resolution
	// -i "0": Default video device
	// -f mjpeg: Output format
	// -: Output to stdout

	fps := 30

	cmd := exec.Command(
		"ffmpeg",
		"-f", "avfoundation",
		"-framerate", fmt.Sprintf("%d", fps),
		"-video_size", fmt.Sprintf("%dx%d", c.width, c.height),
		"-i", "0", // Device 0 = default camera
		"-f", "mjpeg", // MJPEG output stream
		"-q:v", "5", // Quality
		"-hide_banner",       // Hide ffmpeg banner
		"-loglevel", "error", // Only show errors
		"-", // Output to stdout
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	streamingCmd = cmd
	StreamingPipe = stdout
	slog.Info("Started camera streaming process (ffmpeg)", "width", c.width, "height", c.height, "fps", fps)

	// Start consuming frames in background
	go PumpFrames()

	// Start a goroutine to monitor the process and clean up if it exits
	go func() {
		err := cmd.Wait()
		slog.Warn("Camera streaming process exited", "error", err)
		CmdMutex.Lock()
		defer CmdMutex.Unlock()
		streamingCmd = nil
		StreamingPipe = nil
	}()

	return nil
}

// listMacCameras lists available camera devices on macOS
func listMacCameras() (string, error) {
	// ... (unchanged)
	return "", nil
}
