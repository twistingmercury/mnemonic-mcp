package patterns

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupHandlers associates the handlers related to patterns endpoints
// to the gin.Engine that is passed in.
func SetupHandlers(r *gin.Engine) {
	r.GET("/api/patterns", ListPatterns)
	r.POST("/api/patterns", CreatePattern)
	r.GET("/api/patterns/:id", GetPattern)
	r.PUT("/api/patterns/:id", UpdatePattern)
	r.DELETE("/api/patterns/:id", DeletePattern)
}

// ListPatterns handles GET /api/patterns
// See api/openapi/mnemonic-v1.yaml:1775
func ListPatterns(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// CreatePattern handles POST /api/patterns
// See api/openapi/mnemonic-v1.yaml:1831
func CreatePattern(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// GetPattern handles GET /api/patterns/{id}
// See api/openapi/mnemonic-v1.yaml:1896
func GetPattern(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// UpdatePattern handles PUT /api/patterns/{id}
// See api/openapi/mnemonic-v1.yaml:1938
func UpdatePattern(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// DeletePattern handles DELETE /api/patterns/{id}
// See api/openapi/mnemonic-v1.yaml:1972
func DeletePattern(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
