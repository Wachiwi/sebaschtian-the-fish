package handlers

import (
	"embed"
	"log/slog"
	"net/http"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/playlist"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	queueDepthGauge metric.Int64Gauge
)

func init() {
	var err error
	meter := otel.Meter("github.com/wachiwi/sebaschtian-the-fish/cmd/sounds")
	queueDepthGauge, err = meter.Int64Gauge("sounds.queue.depth",
		metric.WithDescription("Current number of items in the queue"),
		metric.WithUnit("{items}"),
	)
	if err != nil {
		slog.Error("Failed to create queue metrics", "error", err)
	}
}

type QueueHandler struct {
	TemplateFS embed.FS
}

func (h *QueueHandler) List(c *gin.Context) {
	queueItems, err := playlist.GetQueueItems()
	if err != nil {
		slog.Error("Failed to get queue items", "error", err)
		c.String(http.StatusInternalServerError, "Failed to get queue")
		return
	}

	tmpl := template.Must(template.New("sounds.html").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).ParseFS(h.TemplateFS, "templates/sounds.html"))
	tmpl.ExecuteTemplate(c.Writer, "queue-list", gin.H{"queueItems": queueItems})
}

func (h *QueueHandler) Play(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.String(http.StatusBadRequest, "Filename required")
		return
	}

	// Add to queue
	item := playlist.QueueItem{
		Name: filename,
		Type: "song",
	}
	if err := playlist.AddToQueue(item); err != nil {
		slog.Error("Failed to add to queue", "error", err)
		c.String(http.StatusInternalServerError, "Failed to queue playback")
		return
	}

	slog.Info("Queued playback", "filename", filename)

	// Return updated queue list
	queueItems, err := playlist.GetQueueItems()
	if err != nil {
		slog.Error("Failed to get queue items", "error", err)
		c.String(http.StatusInternalServerError, "Failed to get queue")
		return
	}
	queueDepthGauge.Record(c.Request.Context(), int64(len(queueItems)))

	tmpl := template.Must(template.New("sounds.html").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).ParseFS(h.TemplateFS, "templates/sounds.html"))
	tmpl.ExecuteTemplate(c.Writer, "queue-list", gin.H{"queueItems": queueItems})
}
