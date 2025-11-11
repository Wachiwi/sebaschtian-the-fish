package main

import (
	"embed"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
)

type SoundFile struct {
	Name string
	Path string
}

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

func GetSoundFiles() []SoundFile {
	dataDir := "./sound-data"
	files, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				log.Printf("Failed to create data directory: %v", err)
				return []SoundFile{}
			}
			return []SoundFile{}
		}
		log.Printf("Failed to read data directory: %v", err)
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
	gin.SetMode(gin.ReleaseMode)
	// --- Authentication Setup ---
	user := os.Getenv("SOUNDS_USER")
	password := os.Getenv("SOUNDS_PASSWORD")
	if user == "" || password == "" {
		log.Fatal("SOUNDS_USER and SOUNDS_PASSWORD environment variables must be set")
	}
	authAccounts := gin.Accounts{
		user: password,
	}

	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1"})

	// --- Public Routes ---
	// Static assets like the logo are public.
	staticSubFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("Failed to create static sub-filesystem: %v", err)
	}
	router.GET("/static/*filepath", func(c *gin.Context) {
		c.FileFromFS(c.Param("filepath"), http.FS(staticSubFS))
	})

	// The API for the fish is public.
	router.GET("/api/random-sound", func(c *gin.Context) {
		soundFiles := GetSoundFiles()
		if len(soundFiles) == 0 {
			c.String(http.StatusNotFound, "No sound files available")
			return
		}
		rand.Seed(time.Now().UnixNano())
		randomSound := soundFiles[rand.Intn(len(soundFiles))]
		filePath := filepath.Join("./sound-data", randomSound.Name)
		contentType := "application/octet-stream"
		ext := filepath.Ext(randomSound.Name)
		if ext == ".wav" || ext == ".WAV" {
			contentType = "audio/wav"
		} else if ext == ".mp3" {
			contentType = "audio/mpeg"
		}
		c.Header("Content-Type", contentType)
		c.File(filePath)
	})

	// --- Authenticated Routes ---
	authorized := router.Group("/", gin.BasicAuth(authAccounts))

	// The sound files themselves require authentication to play directly.
	authorized.Static("/sounds", "./sound-data")

	// The main UI requires authentication.
	authorized.GET("/", func(c *gin.Context) {
		soundFiles := GetSoundFiles()
		tmpl := template.Must(template.ParseFS(templateFS, "templates/sounds.html"))
		err := tmpl.Execute(c.Writer, gin.H{"soundFiles": soundFiles})
		if err != nil {
			log.Printf("Template execution error: %v", err)
			c.String(http.StatusInternalServerError, "Failed to render page")
		}
	})

	// Uploading files requires authentication.
	authorized.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("soundFile")
		if err != nil {
			c.String(http.StatusBadRequest, "Bad request: %v", err)
			return
		}
		dst := filepath.Join("./sound-data", file.Filename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.String(http.StatusInternalServerError, "Failed to save file: %v", err)
			return
		}
		soundFiles := GetSoundFiles()
		tmpl := template.Must(template.ParseFS(templateFS, "templates/sounds.html"))
		err = tmpl.ExecuteTemplate(c.Writer, "sound-list", gin.H{"soundFiles": soundFiles})
		if err != nil {
			log.Printf("Template fragment execution error: %v", err)
			c.String(http.StatusInternalServerError, "Failed to render fragment")
		}
	})

	log.Println("Server is running on http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
