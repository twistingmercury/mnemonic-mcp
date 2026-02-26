package main

import (
	"log"
	"os"

	"github.com/spf13/pflag"
	"github.com/twistingmercury/mnemonic/internal/config"
	"github.com/twistingmercury/mnemonic/internal/server"
	"github.com/twistingmercury/mnemonic/internal/version"
)

var verFlag = pflag.Bool("version", false, "Displays current version information for mnemonic")
var healthFlag = pflag.Bool("health", false, "Get the current health of the service")

func main() {
	pflag.Parse()

	if *verFlag {
		println(version.Print())
		os.Exit(0)
	}

	if *healthFlag {
		exitCode := checkHealth()
		os.Exit(exitCode)
	}

	cfg, err := config.Load()

	if err != nil {
		log.Fatalf("failed to load configuration: %s", err)
	}

	// Health checks are initialized inside ListenAndServe after database
	// connections are established, so no separate health.Initialize call
	// is needed here.

	if err := server.ListenAndServe(cfg); err != nil {
		log.Fatalf("exited with err: %s\n", err.Error())
	}
}

func checkHealth() (exitCode int) {
	exitCode = 0
	//--> TODO: this will need to be implemented in Phase 19
	// The CLI health check requires opening database connections to ping
	// dependencies. For now, this is a placeholder that always returns 0.
	// <--

	return
}
