package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/twistingmercury/heartbeat"
	"github.com/twistingmercury/mnemonic/cmd/version"
)

// SetupHandlers associates the handlers related to operations endpoints
// to the gin.Engine that is passed in.
func SetupHandlers(r *gin.Engine) {
	deps := DefineDependencies()

	r.GET("/ops/health", heartbeat.Handler("mnemonic", deps...))
	r.GET("/ops/version", GetVersion)
}

// GetVersion handles GET /version
// See api/openapi/mnemonic-v1.yaml:2276
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":    "mnemonic",
		"version":    version.Version(),
		"build_date": version.BuildDate(),
		"commit":     version.Commit(),
	})
}

func DefineDependencies() []heartbeat.DependencyDescriptor {
	deps := []heartbeat.DependencyDescriptor{
		{
			Connection: "https://golang.org/",
			Name:       "Golang Site",
			Type:       "Website",
		}}

	return deps
}
