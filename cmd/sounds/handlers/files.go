package handlers

import (
	"embed"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/playlist"
)

type SoundFile struct {
	Name string
	Path string
}

type FileHandler struct {
	TemplateFS embed.FS
}

func GetSoundFiles() []SoundFile {
	dataDir := "./sound-data"
	files, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				slog.Error("Failed to create data directory", "error", err)
			}
			return []SoundFile{}
		}
		slog.Error("Failed to read data directory", "error", err)
		return []SoundFile{}
	}

	var soundFiles []SoundFile
	for _, file := range files {
		if !file.IsDir() && (filepath.Ext(file.Name()) == ".mp3" || filepath.Ext(file.Name()) == ".wav" || filepath.Ext(file.Name()) == ".json") {
			// Skip JSON files from sound list, but read audio
			if filepath.Ext(file.Name()) != ".json" {
				soundFiles = append(soundFiles, SoundFile{
					Name: file.Name(),
					Path: filepath.Join("/sounds", file.Name()),
				})
			}
		}
	}
	return soundFiles
}

func (h *FileHandler) Index(c *gin.Context) {
	soundFiles := GetSoundFiles()
	playedItems, err := playlist.GetPlayedItems()
	if err != nil {
		slog.Error("Failed to get played items", "error", err)
		playedItems = []playlist.PlayedItem{}
	}

	queueItems, err := playlist.GetQueueItems()
	if err != nil {
		slog.Error("Failed to get queue items", "error", err)
		queueItems = []playlist.QueueItem{}
	}

	// Sort by timestamp descending
	sort.Slice(playedItems, func(i, j int) bool {
		return playedItems[i].Timestamp.After(playedItems[j].Timestamp)
	})

	tmpl := template.Must(template.New("sounds.html").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).ParseFS(h.TemplateFS, "templates/sounds.html"))
	err = tmpl.Execute(c.Writer, gin.H{
		"soundFiles":  soundFiles,
		"playedItems": playedItems,
		"queueItems":  queueItems,
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to render page")
	}
}

func (h *FileHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("soundFile")
	if err != nil {
		c.String(http.StatusBadRequest, "Bad request")
		return
	}
	dst := filepath.Join("./sound-data", file.Filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.String(http.StatusInternalServerError, "Failed to save file")
		return
	}
	soundFiles := GetSoundFiles()
	tmpl := template.Must(template.ParseFS(h.TemplateFS, "templates/sounds.html"))
	tmpl.ExecuteTemplate(c.Writer, "sound-list", gin.H{"soundFiles": soundFiles})
}
