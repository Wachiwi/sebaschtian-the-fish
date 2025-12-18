//go:build linux && arm64

package camera

import (
	"bytes"
	"fmt"
	"log/slog"
	"os/exec"
)

var streamingCmd *exec.Cmd

// captureFrameRPi captures a frame using libcamera-apps/rpicam-apps on Raspberry Pi.
// It uses a persistent background process (libcamera-vid) to stream MJPEG frames,
// parsing the output stream to extract individual JPEG images. This avoids the overhead
// of restarting the camera hardware for every frame.
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

// startStreamingProcess starts the libcamera-vid process in MJPEG streaming mode.
func (c *Camera) startStreamingProcess() error {
	// Determine command name (rpicam-vid for newer OS, libcamera-vid for older)
	cmdName := "rpicam-vid"
	if _, err := exec.LookPath(cmdName); err != nil {
		cmdName = "libcamera-vid"
		if _, err := exec.LookPath(cmdName); err != nil {
			return fmt.Errorf("neither rpicam-vid nor libcamera-vid found")
		}
	}

	cmd := exec.Command(
		cmdName,
		"--width", fmt.Sprintf("%d", c.width),
		"--height", fmt.Sprintf("%d", c.height),
		"--timeout", "0", // Run indefinitely
		"--nopreview",
		"--codec", "mjpeg", // MJPEG output
		"--output", "-", // Output to stdout
		"--framerate", fmt.Sprintf("%d", c.fps),
		// Module 3 specific optimizations
		"--awb", "auto",
		"--metering", "average",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Capture stderr for debugging
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w, stderr: %s", cmdName, err, stderr.String())
	}

	streamingCmd = cmd
	StreamingPipe = stdout
	slog.Info("Started camera streaming process", "command", cmdName, "width", c.width, "height", c.height, "fps", c.fps)

	// Start consuming frames in background
	go PumpFrames()

	// Start a goroutine to monitor the process and clean up if it exits
	go func() {
		err := cmd.Wait()
		if err != nil {
			slog.Warn("Camera streaming process exited", "error", err, "stderr", stderr.String())
		} else {
			slog.Info("Camera streaming process exited cleanly")
		}
		CmdMutex.Lock()
		defer CmdMutex.Unlock()
		streamingCmd = nil
		StreamingPipe = nil
	}()

	return nil
}
