package camera

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"sync"
	"time"
)

// Camera represents a Raspberry Pi Camera Module v3
type Camera struct {
	mu             sync.RWMutex
	latestFrame    []byte
	isStreaming    bool
	stopChan       chan struct{}
	width          int
	height         int
	fps            int
	loggedFallback bool // Track if we've logged the fallback message
}

// Config holds camera configuration
type Config struct {
	Width  int
	Height int
	FPS    int
}

// NewCamera creates a new camera instance with the given configuration
func NewCamera(config Config) *Camera {
	if config.Width == 0 {
		config.Width = 640
	}
	if config.Height == 0 {
		config.Height = 480
	}
	if config.FPS == 0 {
		config.FPS = 30
	}

	return &Camera{
		width:    config.Width,
		height:   config.Height,
		fps:      config.FPS,
		stopChan: make(chan struct{}),
	}
}

// Start begins capturing frames from the camera
func (c *Camera) Start() error {
	c.mu.Lock()
	if c.isStreaming {
		c.mu.Unlock()
		return fmt.Errorf("camera is already streaming")
	}
	c.isStreaming = true
	c.mu.Unlock()

	go c.captureLoop()
	log.Printf("Camera started: %dx%d @ %d fps", c.width, c.height, c.fps)
	return nil
}

// Stop stops capturing frames
func (c *Camera) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isStreaming {
		return
	}

	c.isStreaming = false
	close(c.stopChan)
	log.Println("Camera stopped")
}

// GetFrame returns the latest captured frame as JPEG bytes
func (c *Camera) GetFrame() []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.latestFrame == nil {
		return nil
	}

	// Return a copy to prevent race conditions
	frame := make([]byte, len(c.latestFrame))
	copy(frame, c.latestFrame)
	return frame
}

// IsStreaming returns whether the camera is currently streaming
func (c *Camera) IsStreaming() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isStreaming
}

// captureLoop continuously captures frames from the camera
func (c *Camera) captureLoop() {
	ticker := time.NewTicker(time.Second / time.Duration(c.fps))
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			frame, err := c.captureFrame()
			if err != nil {
				log.Printf("Error capturing frame: %v", err)
				continue
			}

			c.mu.Lock()
			c.latestFrame = frame
			c.mu.Unlock()
		}
	}
}

// captureFrame captures a single frame from the camera
// On Raspberry Pi (linux/arm64), this uses libcamera-apps which properly handles
// the Camera Module v3's complex ISP (Image Signal Processor)
// On macOS (darwin), this uses ffmpeg to capture from the built-in webcam
// On other platforms, generates a placeholder frame for testing
func (c *Camera) captureFrame() ([]byte, error) {
	// Try platform-specific implementation first
	// - camera_rpi.go on linux/arm64 (Raspberry Pi Camera Module v3)
	// - camera_darwin.go on macOS (built-in webcam via ffmpeg)
	// - camera_other.go on other platforms (returns error)
	frame, err := c.captureFrameRPi()
	if err == nil {
		return frame, nil
	}

	// Log the error (only once per camera instance to avoid spam)
	c.mu.Lock()
	if !c.loggedFallback {
		log.Printf("Camera capture failed, using placeholder frames: %v", err)
		c.loggedFallback = true
	}
	c.mu.Unlock()

	// Fallback to placeholder for development/testing if camera not available
	return c.generatePlaceholderFrame()
}

// generatePlaceholderFrame creates a simple colored frame for testing
// This should be replaced with actual camera capture in production
func (c *Camera) generatePlaceholderFrame() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, c.width, c.height))

	// Create a simple pattern with timestamp
	timestamp := time.Now().Unix()
	color := byte(timestamp % 256)

	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			offset := y*img.Stride + x*4
			img.Pix[offset] = color
			img.Pix[offset+1] = byte((x * 255) / c.width)
			img.Pix[offset+2] = byte((y * 255) / c.height)
			img.Pix[offset+3] = 255
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return buf.Bytes(), nil
}
