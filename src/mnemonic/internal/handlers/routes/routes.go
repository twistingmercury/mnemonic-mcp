package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupHandlers associates the handlers related to routes endpoints
// to the gin.Engine that is passed in.
func SetupHandlers(r *gin.Engine) {
	r.POST("/api/route", RoutePrompt)
}

// RoutePrompt handles POST /api/route
// See api/openapi/mnemonic-v1.yaml:1391
func RoutePrompt(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
