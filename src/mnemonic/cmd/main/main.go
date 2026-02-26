package main

import (
	"log"
	"os"

	"github.com/spf13/pflag"
	"github.com/twistingmercury/mnemonic/internal/config"
	"github.com/twistingmercury/mnemonic/internal/health"
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

	if err := health.Initialize(cfg); err != nil {
		log.Fatalf("failed to initialize health check: %s", err)
	}

	if err := server.ListenAndServe(cfg); err != nil {
		log.Fatalf("exited with err: %s\n", err.Error())
	}
}

func checkHealth() (exitCode int) {
	exitCode = 0
	//--> TODO: this will need to be implemented in Phase 19
	// cfg, err := config.Load()

	// if err != nil {
	// 	log.Fatalf("failed to load configuration: %s", err)
	// }

	// if err := health.Initialize(cfg); err != nil {
	// 	log.Fatalf("failed to initialize health check: %s", err)
	// }

	// if *healthFlag {
	// 	err := health.CheckHealth()
	// 	if err != nil {
	// 		log.Fatalf("health check failed with this error: %s", err)
	// 		exitCode = 1
	// 	}
	// }
	// <--

	return
}
