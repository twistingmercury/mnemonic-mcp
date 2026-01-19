package main

import (
	"os"

	"github.com/spf13/pflag"
	"github.com/twistingmercury/mnemonic/cmd/version"
)

var verFlag = pflag.Bool("version", false, "Displays current version information for mnemonic")

func main() {
	pflag.Parse()

	if *verFlag {
		println(version.Print())
		os.Exit(0)
	}

	println("this is Mnemonic!")
}
