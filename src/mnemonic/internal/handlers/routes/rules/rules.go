package rules

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupHandlers associates the handlers related to routing rules endpoints
// to the gin.Engine that is passed in.
func SetupHandlers(r *gin.Engine) {
	r.GET("/api/routing-rules", ListRoutingRules)
	r.POST("/api/routing-rules", CreateRoutingRule)
	r.GET("/api/routing-rules/:id", GetRoutingRule)
	r.PUT("/api/routing-rules/:id", UpdateRoutingRule)
	r.DELETE("/api/routing-rules/:id", DeleteRoutingRule)
}

// ListRoutingRules handles GET /api/routing-rules
// See api/openapi/mnemonic-v1.yaml:1995
func ListRoutingRules(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// CreateRoutingRule handles POST /api/routing-rules
// See api/openapi/mnemonic-v1.yaml:2066
func CreateRoutingRule(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// GetRoutingRule handles GET /api/routing-rules/{id}
// See api/openapi/mnemonic-v1.yaml:2138
func GetRoutingRule(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// UpdateRoutingRule handles PUT /api/routing-rules/{id}
// See api/openapi/mnemonic-v1.yaml:2181
func UpdateRoutingRule(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// DeleteRoutingRule handles DELETE /api/routing-rules/{id}
// See api/openapi/mnemonic-v1.yaml:2215
func DeleteRoutingRule(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
