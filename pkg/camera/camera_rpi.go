//go:build linux && arm64

package camera

import (
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
		"--inline",         // Inline headers (SPS/PPS) - critical for streaming
		"--output", "-",    // Output to stdout
		"--framerate", fmt.Sprintf("%d", c.fps),
		// Module 3 specific optimizations
		"--awb", "auto",
		"--metering", "average",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", cmdName, err)
	}

	streamingCmd = cmd
	StreamingPipe = stdout
	slog.Info("Started camera streaming process", "command", cmdName, "width", c.width, "height", c.height, "fps", c.fps)

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

	}

	// 2. Read from the stream until we get a full JPEG frame
	return c.readNextFrameFromStream()
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
		"--inline",      // Inline headers (SPS/PPS) - critical for streaming
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

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", cmdName, err)
	}

	streamingCmd = cmd
	streamingPipe = stdout
	slog.Info("Started camera streaming process", "command", cmdName, "width", c.width, "height", c.height, "fps", c.fps)

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


// readNextFrameFromStream parses the MJPEG stream to find the next JPEG image.
// JPEG Start of Image (SOI): FF D8
// JPEG End of Image (EOI): FF D9
func (c *Camera) readNextFrameFromStream() ([]byte, error) {
	if streamingPipe == nil {
		return nil, fmt.Errorf("streaming pipe is nil")
	}

	// Buffer to build the current frame
	const bufferSize = 4096
	readBuf := make([]byte, bufferSize)
	var frameBuf bytes.Buffer
	inFrame := false

	// We need a timeout to prevent hanging if the camera stops sending data
	// Note: Since we are inside a mutex lock here (from captureFrameRPi),
	// this timeout effectively blocks other readers.
	// In a production parsing loop, this would be a continuous background goroutine
	// updating a shared 'latestFrame' atomic value.
	// For this refactor, we are doing synchronous reading to match existing architecture,
	// but we must be careful.

	// Ideally, we shouldn't be reading just ONE frame here. We should probably be
	// buffering the stream in the background. But let's try to find ONE frame.

	// Problem: If we just read bytes, we might miss the start if we are out of sync.
	// But since this function is called repeatedly in a loop by Camera.captureLoop,
	// it expects to return *a* frame.

	// Better approach for `captureFrameRPi` with persistent process:
	// It's hard to robustly read exactly one frame on demand from a continuous pipe without buffering.
	// If we read partial data, the next call needs to resume.
	//
	// REFACTOR DECISION:
	// Instead of `captureFrameRPi` returning a frame on demand, `startStreamingProcess`
	// should start a goroutine that continuously reads the pipe and updates `c.latestFrame`.
	// Then `captureFrameRPi` simply returns `c.latestFrame` (or waits for a new one).

	// However, `captureFrameRPi` matches the interface of `captureFrame` called by `captureLoop`.
	// `captureLoop` runs on a ticker.
	// If we change the architecture to "push" updates from the stream parser, we might not need `captureLoop`.

	// Let's stick to the simplest fix:
	// We will implement a specialized reader here that reads until it finds a full JPEG.
	// BUT: Since we can't easily push back bytes into the pipe, we might lose data between calls
	// if we don't manage state.
	// The `streamingPipe` is an `io.ReadCloser`.

	// Actually, the `captureLoop` in `camera.go` is:
	// ticker -> captureFrame() -> c.latestFrame = frame
	//
	// If we move the reading logic to a background goroutine started by `startStreamingProcess`,
	// we can just make `captureFrameRPi` return the current value of that background buffer.

	// Let's modify the plan:
	// 1. `startStreamingProcess` starts a goroutine `pumpFrames`.
	// 2. `pumpFrames` continuously reads stdout, parses JPEGs, and updates a package-level `currentFrame` variable (protected by mutex).
	// 3. `captureFrameRPi` just returns a copy of `currentFrame`.

	return getCurrentFrame()
}

var (
	currentFrame      []byte
	currentFrameMutex sync.RWMutex
	lastFrameTime     time.Time
)

func getCurrentFrame() ([]byte, error) {
	currentFrameMutex.RLock()
	defer currentFrameMutex.RUnlock()
	if len(currentFrame) == 0 {
		return nil, fmt.Errorf("no frame available yet")
	}
	// Check if frame is stale (e.g. process died but variable wasn't cleared)
	if time.Since(lastFrameTime) > 5*time.Second {
		return nil, fmt.Errorf("frame is stale (>5s old)")
	}

	// Return copy
	dst := make([]byte, len(currentFrame))
	copy(dst, currentFrame)
	return dst, nil
}

func (c *Camera) pumpFrames() {
	if streamingPipe == nil {
		return
	}

	// Scanner that splits on JPEG boundaries would be ideal, but standard bufio.Scanner
	// doesn't support complex byte sequence splitting easily.
	// We'll read chunks and assemble manually.

	const readChunkSize = 4096
	buf := make([]byte, readChunkSize)
	var frameBuffer []byte

	// JPEG markers
	soi := []byte{0xFF, 0xD8}
	eoi := []byte{0xFF, 0xD9}

	for {
		n, err := streamingPipe.Read(buf)
		if err != nil {
			slog.Error("Stream read error", "error", err)
			return
		}
		if n == 0 {
			continue
		}

		chunk := buf[:n]

		// This is a naive parser. It assumes standard MJPEG structure.
		// It appends data to frameBuffer.
		// If it finds SOI, it resets buffer (if we were looking for start).
		// If it finds EOI, it finishes the frame.

		// Simplified logic:
		// 1. If we don't have a start, look for SOI.
		// 2. If we have a start, append.
		// 3. Look for EOI. If found, save frame and reset.

		// Optimization: Search in the current chunk

		// If buffer is empty, we are looking for start
		startIndex := -1
		if len(frameBuffer) == 0 {
			startIndex = bytes.Index(chunk, soi)
			if startIndex == -1 {
				continue // No start found, discard chunk
			}
		}

		if startIndex != -1 {
			// Found start!
			frameBuffer = append(frameBuffer, chunk[startIndex:]...)
			// Adjust chunk to start search for EOI after the SOI
			chunk = chunk[startIndex:]
		} else {
			// Already inside a frame, append everything
			frameBuffer = append(frameBuffer, chunk...)
		}

		// Look for End of Image in the *appended* data (or just the new chunk to be efficient)
		// Careful: EOI might be split across chunks (FF in one, D9 in next).
		// For simplicity, we check the tail of frameBuffer.

		// We only check if we have enough data
		if len(frameBuffer) > 2 {
			// Check for EOI in the newly added data (plus one byte back in case of split)
			// Actually, let's just search the last (readChunkSize + safety) bytes
			searchWindow := frameBuffer
			if len(searchWindow) > 2*readChunkSize {
				searchWindow = frameBuffer[len(frameBuffer)-readChunkSize-10:]
			}

			eoiIndex := bytes.Index(searchWindow, eoi)
			if eoiIndex != -1 {
				// We found the end!
				// The index is relative to searchWindow.
				// We need the absolute index in frameBuffer.

				// Let's rely on standard bytes.LastIndex on the whole buffer for correctness first, optimize later.
				// MJPEG streams shouldn't have embedded thumbnails with EOI usually.

				realEOIIndex := bytes.LastIndex(frameBuffer, eoi)

				if realEOIIndex != -1 {
					// Extract full frame
					fullFrame := frameBuffer[:realEOIIndex+2]

					// Update global state
					currentFrameMutex.Lock()
					currentFrame = make([]byte, len(fullFrame))
					copy(currentFrame, fullFrame)
					lastFrameTime = time.Now()
					currentFrameMutex.Unlock()

					// Reset buffer.
					// IMPORTANT: If there is data AFTER EOI in this chunk, it is the start of the next frame!
					// We need to keep it.
					if len(frameBuffer) > realEOIIndex+2 {
						remaining := frameBuffer[realEOIIndex+2:]
						frameBuffer = make([]byte, len(remaining))
						copy(frameBuffer, remaining)
					} else {
						frameBuffer = nil
					}
				}
			}
		}

		// Safety: prevent buffer from growing indefinitely if no EOI found
		if len(frameBuffer) > 10*1024*1024 { // 10MB limit
			frameBuffer = nil
			slog.Warn("Frame buffer overflow, resetting")
		}
	}
}
