package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// AuthRequired is a middleware to check for a valid session.
func AuthRequired(c *gin.Context) {
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
