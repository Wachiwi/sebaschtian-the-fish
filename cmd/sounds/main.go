package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/gin-gonic/gin"
)

type SoundFile struct {
	Name string
	Path string
}

//go:embed all:templates
var templates embed.FS

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
	router := gin.Default()

	router.Static("/sounds", "./sound-data")

	router.GET("/", func(c *gin.Context) {
		soundFiles := GetSoundFiles()
		tmpl := template.Must(template.ParseFS(templates, "templates/sounds.html"))
		err := tmpl.Execute(c.Writer, gin.H{"soundFiles": soundFiles})
		if err != nil {
			log.Printf("Template execution error: %v", err)
			c.String(http.StatusInternalServerError, "Failed to render page")
		}
	})

	router.POST("/upload", func(c *gin.Context) {
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
		tmpl := template.Must(template.ParseFS(templates, "templates/sounds.html"))
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
