package main

import (
	"log"
	"os"

	"github.com/spf13/pflag"
	"github.com/twistingmercury/mnemonic/cmd/version"
	"github.com/twistingmercury/mnemonic/internal/server"
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
		err := server.CheckHealth()
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	if err := server.ListenAndServe(); err != nil {
		log.Printf("exited with err: %s\n", err.Error())
	}
}
