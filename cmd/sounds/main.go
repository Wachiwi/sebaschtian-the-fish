package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/wachiwi/sebaschtian-the-fish/cmd/sounds/handlers"
	"github.com/wachiwi/sebaschtian-the-fish/cmd/sounds/middleware"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/camera"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/logger"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/playlist"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

func main() {
	logger.Setup()

	// Initialize Playlist with the correct path
	// The container mounts the volume at /app/sound-data, which corresponds to ./sound-data relative to WORKDIR /app
	playlist.Init("./sound-data")

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

	// --- Handlers Setup ---
	authHandler := &handlers.AuthHandler{
		User:       user,
		Password:   password,
		TemplateFS: templateFS,
	}
	fileHandler := &handlers.FileHandler{TemplateFS: templateFS}
	queueHandler := &handlers.QueueHandler{TemplateFS: templateFS}
	cameraHandler := &handlers.CameraHandler{Cam: cam}

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

	router.GET("/login", authHandler.LoginPage)
	router.POST("/login", authHandler.Login)

	// --- Authenticated Routes ---
	authorized := router.Group("/")
	authorized.Use(middleware.AuthRequired)
	{
		authorized.Static("/sounds", "./sound-data")

		authorized.GET("/", fileHandler.Index)
		authorized.POST("/upload", fileHandler.Upload)
		authorized.GET("/queue", queueHandler.List)
		authorized.POST("/play/:filename", queueHandler.Play)
		authorized.GET("/logout", authHandler.Logout)
		authorized.GET("/camera/stream", cameraHandler.Stream)
	}

	slog.Info("Server is running", "url", fmt.Sprintf("http://localhost:%s", port))
	router.Run(fmt.Sprintf(":%s", port))
}
