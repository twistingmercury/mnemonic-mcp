package health

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/twistingmercury/heartbeat"
	"github.com/twistingmercury/mnemonic/internal/config"
)

var cfg *config.MnemonicConfig
var deps []heartbeat.DependencyDescriptor

func Initialize(conf *config.MnemonicConfig) error {
	if conf == nil {
		return errors.New("config.MnemonicConfig is nil")
	}
	cfg = conf

	deps = []heartbeat.DependencyDescriptor{
		{
			Name:        "PostgreSQL check",
			Type:        "database",
			HandlerFunc: checkPostgreSQLHealth,
		},
		{
			Name:        "Neo4j check",
			Type:        "database",
			HandlerFunc: checkNeo4jHealth,
		},
		{
			Name:        "OpenAI embedding model check",
			Type:        "embedding model",
			HandlerFunc: checkEmbeddingModel,
		},
		{
			Name:        "OpenAI extraction model check",
			Type:        "extraction model",
			HandlerFunc: checkExtractionModel,
		},
	}

	return nil
}

func CheckHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, depResults := heartbeat.CheckDependencies(ctx, deps)

	var errs error
	for _, ds := range depResults {
		if ds.Status != heartbeat.StatusOK {
			err := fmt.Errorf("failed to verify health for %s: %s", ds.Resource, ds.Message)
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

// CheckPostgreSQLHealth verifies PostgreSQL connectivity by running a simple query.
func checkPostgreSQLHealth() (status heartbeat.StatusResult) {
	connection := cfg.Database.Postgres.ConnectionString()

	// TODO: Remove this if statement...it's just a placeholder to be able to use cfg var.
	if len(connection) == 0 {
		status.Status = heartbeat.StatusCritical
		status.Message = "no connection string found"
	}
	// TODO: checkPostgreSQLHealth needs implementing
	return status
}

// CheckNeo4jHealth verifies Neo4j connectivity by running a simple query.
func checkNeo4jHealth() (status heartbeat.StatusResult) {
	// connection := cfg.Database.Neo4j.
	// TODO: checkNeo4jHealth needs implementing
	return
}

func checkEmbeddingModel() (status heartbeat.StatusResult) {
	// TODO: checkEmbeddingModel needs implementing
	status.Status = heartbeat.StatusNotSet
	return
}

func checkExtractionModel() (status heartbeat.StatusResult) {
	// TODO: checkExtractionModel needs implementing
	status.Status = heartbeat.StatusNotSet
	return
}
