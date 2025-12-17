package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/camera"
)

type CameraHandler struct {
	Cam *camera.Camera
}

func (h *CameraHandler) Stream(c *gin.Context) {
	if !h.Cam.IsStreaming() {
		c.String(http.StatusServiceUnavailable, "Camera not available")
		return
	}

	c.Header("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Stream frames continuously
	ticker := time.NewTicker(16 * time.Millisecond) // ~60 fps
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			frame := h.Cam.GetFrame()
			if frame == nil {
				continue
			}

			// Write MJPEG frame
			fmt.Fprintf(w, "--frame\r\n")
			fmt.Fprintf(w, "Content-Type: image/jpeg\r\n")
			fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(frame))
			w.Write(frame)
			fmt.Fprintf(w, "\r\n")
			flusher.Flush()
		}
	}
}
