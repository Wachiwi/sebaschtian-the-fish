package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/camera"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/logger"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/playlist"
	"sort"
	"time"
)

type SoundFile struct {
	Name string
	Path string
}

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

// authRequired is a middleware to check for a valid session.
func authRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil {
		// If the request is from HTMX, trigger a client-side redirect.
		if c.GetHeader("HX-Request") == "true" {
			c.Header("HX-Redirect", "/login")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Otherwise, do a standard server-side redirect.
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
		return
	}
	c.Next()
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
		if !file.IsDir() {
			soundFiles = append(soundFiles, SoundFile{
				Name: file.Name(),
				Path: filepath.Join("/sounds", file.Name()),
			})
		}
	}
	return soundFiles
}

func main() {
	logger.Setup()
	gin.SetMode(gin.ReleaseMode)
	// --- Credentials and Session Setup ---
	port := os.Getenv("SOUNDS_PORT")
	if port == "" {
		port = "8080"
	}

	user := os.Getenv("SOUNDS_USER")
	password := os.Getenv("SOUNDS_PASSWORD")
	sessionSecret := os.Getenv("SOUNDS_SESSION_SECRET")
	if user == "" || password == "" || sessionSecret == "" {
		logger.Fatal("SOUNDS_USER, SOUNDS_PASSWORD, and SOUNDS_SESSION_SECRET must be set")
	}

	// --- Camera Setup ---
	// RPi Camera Module v3 specs: up to 2304x1296 @ 56fps or 1920x1080 @ 120fps
	// Using 1920x1080 @ 60fps for high quality, smooth video
	cam := camera.NewCamera(camera.Config{
		Width:  1920,
		Height: 1080,
		FPS:    60,
	})
	if err := cam.Start(); err != nil {
		slog.Warn("Failed to start camera", "error", err)
	}
	defer cam.Stop()

	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1"})

	store := cookie.NewStore([]byte(sessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("sound_session", store))

	// --- Public Routes ---
	staticSubFS, _ := fs.Sub(staticFS, "static")
	router.GET("/static/*filepath", func(c *gin.Context) {
		c.FileFromFS(c.Param("filepath"), http.FS(staticSubFS))
	})

	// Login routes
	router.GET("/login", func(c *gin.Context) {
		tmpl := template.Must(template.ParseFS(templateFS, "templates/login.html"))
		tmpl.Execute(c.Writer, nil)
	})

	router.POST("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		formUser := c.PostForm("username")
		formPassword := c.PostForm("password")

		if formUser == user && formPassword == password {
			session.Set("user", user)
			if err := session.Save(); err != nil {
				slog.Error("Failed to save session", "error", err)
				c.String(http.StatusInternalServerError, "Failed to save session")
				return
			}
			if c.GetHeader("HX-Request") == "true" {
				c.Header("HX-Redirect", "/")
				return
			}
			// Otherwise, do a standard server-side redirect.
			c.Redirect(http.StatusFound, "/")
			return
		} else {
			tmpl := template.Must(template.ParseFS(templateFS, "templates/login.html"))
			tmpl.Execute(c.Writer, gin.H{"error": "Invalid credentials"})
		}
	})

	// --- Authenticated Routes ---
	authorized := router.Group("/")
	authorized.Use(authRequired)
	{
		authorized.Static("/sounds", "./sound-data")

		authorized.GET("/", func(c *gin.Context) {
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
			}).ParseFS(templateFS, "templates/sounds.html"))
			err = tmpl.Execute(c.Writer, gin.H{
				"soundFiles":  soundFiles,
				"playedItems": playedItems,
				"queueItems":  queueItems,
			})
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to render page")
			}
		})

		authorized.POST("/upload", func(c *gin.Context) {
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
			tmpl := template.Must(template.ParseFS(templateFS, "templates/sounds.html"))
			tmpl.ExecuteTemplate(c.Writer, "sound-list", gin.H{"soundFiles": soundFiles})
		})

		authorized.GET("/queue", func(c *gin.Context) {
			queueItems, err := playlist.GetQueueItems()
			if err != nil {
				slog.Error("Failed to get queue items", "error", err)
				c.String(http.StatusInternalServerError, "Failed to get queue")
				return
			}

			tmpl := template.Must(template.New("sounds.html").Funcs(template.FuncMap{
				"add": func(a, b int) int { return a + b },
			}).ParseFS(templateFS, "templates/sounds.html"))
			tmpl.ExecuteTemplate(c.Writer, "queue-list", gin.H{"queueItems": queueItems})
		})

		authorized.POST("/play/:filename", func(c *gin.Context) {
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

			tmpl := template.Must(template.New("sounds.html").Funcs(template.FuncMap{
				"add": func(a, b int) int { return a + b },
			}).ParseFS(templateFS, "templates/sounds.html"))
			tmpl.ExecuteTemplate(c.Writer, "queue-list", gin.H{"queueItems": queueItems})
		})

		authorized.GET("/logout", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Clear()
			session.Save()
			c.Redirect(http.StatusFound, "/login")
		})

		// Camera stream endpoint - MJPEG stream
		authorized.GET("/camera/stream", func(c *gin.Context) {
			if !cam.IsStreaming() {
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
					frame := cam.GetFrame()
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
		})
	}

	slog.Info("Server is running", "url", fmt.Sprintf("http://localhost:%s", port))
	router.Run(fmt.Sprintf(":%s", port))
}
