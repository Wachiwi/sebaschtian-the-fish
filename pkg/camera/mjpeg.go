package camera

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

var (
	// These variables manage the shared streaming process state.
	// They are populated by the platform-specific startStreamingProcess implementations
	// and consumed by the shared pumpFrames logic.
	StreamingPipe io.ReadCloser
	CmdMutex      sync.Mutex

	// Frame state
	currentFrame      []byte
	currentFrameMutex sync.RWMutex
	lastFrameTime     time.Time
)

// GetCurrentFrame returns the latest fully parsed JPEG frame.
// It is safe for concurrent use.
func GetCurrentFrame() ([]byte, error) {
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

// PumpFrames continuously reads MJPEG data from the StreamingPipe,
// extracts JPEG frames, and updates the CurrentFrame.
func PumpFrames() {
	if StreamingPipe == nil {
		return
	}

	const readChunkSize = 4096
	buf := make([]byte, readChunkSize)
	var frameBuffer []byte

	// JPEG markers
	soi := []byte{0xFF, 0xD8}
	eoi := []byte{0xFF, 0xD9}

	for {
		n, err := StreamingPipe.Read(buf)
		if err != nil {
			slog.Error("Stream read error", "error", err)
			return
		}
		if n == 0 {
			continue
		}

		chunk := buf[:n]

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
		if len(frameBuffer) > 2 {
			// Check for EOI in the newly added data (plus one byte back in case of split)
			searchWindow := frameBuffer
			if len(searchWindow) > 2*readChunkSize {
				searchWindow = frameBuffer[len(frameBuffer)-readChunkSize-10:]
			}

			eoiIndex := bytes.Index(searchWindow, eoi)
			if eoiIndex != -1 {
				// We found the end!
				// We need the absolute index in frameBuffer.
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
