package handlers

import (
	"embed"
	"log/slog"
	"net/http"
	"text/template"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	User       string
	Password   string
	TemplateFS embed.FS
}

func (h *AuthHandler) LoginPage(c *gin.Context) {
	tmpl := template.Must(template.ParseFS(h.TemplateFS, "templates/login.html"))
	tmpl.Execute(c.Writer, nil)
}

func (h *AuthHandler) Login(c *gin.Context) {
	session := sessions.Default(c)
	formUser := c.PostForm("username")
	formPassword := c.PostForm("password")

	if formUser == h.User && formPassword == h.Password {
		session.Set("user", h.User)
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
		tmpl := template.Must(template.ParseFS(h.TemplateFS, "templates/login.html"))
		tmpl.Execute(c.Writer, gin.H{"error": "Invalid credentials"})
	}
}

func (h *AuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/login")
}
