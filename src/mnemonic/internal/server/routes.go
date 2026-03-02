package server

import (
	"github.com/gin-gonic/gin"
	agenthandler "github.com/twistingmercury/mnemonic/internal/handlers/agents"
	patternhandler "github.com/twistingmercury/mnemonic/internal/handlers/patterns"
	skillfilehandler "github.com/twistingmercury/mnemonic/internal/handlers/skillfiles"
	skillhandler "github.com/twistingmercury/mnemonic/internal/handlers/skills"
	chunkrepo "github.com/twistingmercury/mnemonic/internal/repository/chunk"
	agentsvc "github.com/twistingmercury/mnemonic/internal/service/agent"
	patternsvc "github.com/twistingmercury/mnemonic/internal/service/pattern"
	searchsvc "github.com/twistingmercury/mnemonic/internal/service/search"
	skillsvc "github.com/twistingmercury/mnemonic/internal/service/skill"
	skillfilesvc "github.com/twistingmercury/mnemonic/internal/service/skillfile"
)

// Services groups all domain services required by the REST API handlers.
type Services struct {
	Agent     agentsvc.Service
	Pattern   patternsvc.Service
	Search    searchsvc.Service
	Skill     skillsvc.Service
	SkillFile skillfilesvc.Service
	ChunkRepo chunkrepo.Repository
}

// RegisterAPIRoutes creates all domain handlers and registers their routes
// on the /v1/api route group. Call this after setting up middleware.
func RegisterAPIRoutes(router *gin.Engine, svc Services) {
	v1 := router.Group("/v1/api")

	agenthandler.New(svc.Agent).RegisterRoutes(v1)
	patternhandler.New(svc.Pattern, svc.Search, svc.ChunkRepo).RegisterRoutes(v1)
	skillhandler.New(svc.Skill).RegisterRoutes(v1)
	skillfilehandler.New(svc.SkillFile).RegisterRoutes(v1)
}
