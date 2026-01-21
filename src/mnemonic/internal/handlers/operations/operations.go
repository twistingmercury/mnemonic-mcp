package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupHandlers associates the handlers related to operations endpoints
// to the gin.Engine that is passed in.
func SetupHandlers(r *gin.Engine) {
	r.GET("/ops/health", HealthCheck)
	r.GET("/ops/version", GetVersion)
}

// HealthCheck handles GET /health
// See api/openapi/mnemonic-v1.yaml:2238
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// GetVersion handles GET /version
// See api/openapi/mnemonic-v1.yaml:2276
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
