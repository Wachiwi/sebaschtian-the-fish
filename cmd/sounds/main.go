package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
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
				log.Printf("Failed to create data directory: %v", err)
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
	// --- Credentials and Session Setup ---
	port := os.Getenv("SOUNDS_PORT")
	if port == "" {
		port = "8080"
	}

	user := os.Getenv("SOUNDS_USER")
	password := os.Getenv("SOUNDS_PASSWORD")
	sessionSecret := os.Getenv("SOUNDS_SESSION_SECRET")
	if user == "" || password == "" || sessionSecret == "" {
		log.Fatal("SOUNDS_USER, SOUNDS_PASSWORD, and SOUNDS_SESSION_SECRET must be set")
	}

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
				log.Printf("Failed to save session: %v", err)
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
			tmpl := template.Must(template.ParseFS(templateFS, "templates/sounds.html"))
			err := tmpl.Execute(c.Writer, gin.H{"soundFiles": soundFiles})
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

		authorized.GET("/logout", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Clear()
			session.Save()
			c.Redirect(http.StatusFound, "/login")
		})
	}

	log.Printf("Server is running on http://localhost:%s\n", port)
	router.Run(fmt.Sprintf(":%s", port))
}
