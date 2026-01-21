package agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupHandlers associates the handlers related to agent endpoints
// to the gin.Engine that is passed in.
func SetupHandlers(r *gin.Engine) {
	r.GET("/api/agents", ListAgents)
	r.POST("/api/agents", CreateAgent)
	r.GET("/api/agents/:name", GetAgent)
	r.PUT("/api/agents/:name", UpdateAgent)
	r.DELETE("/api/agents/:name", DeleteAgent)
}

// ListAgents handles GET /api/agents
// See api/openapi/mnemonic-v1.yaml:1515
func ListAgents(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// CreateAgent handles POST /api/agents
// See api/openapi/mnemonic-v1.yaml:1565
func CreateAgent(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// GetAgent handles GET /api/agents/{name}
// See api/openapi/mnemonic-v1.yaml:1638
func GetAgent(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// UpdateAgent handles PUT /api/agents/{name}
// See api/openapi/mnemonic-v1.yaml:1686
func UpdateAgent(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// DeleteAgent handles DELETE /api/agents/{name}
// See api/openapi/mnemonic-v1.yaml:1735
func DeleteAgent(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
