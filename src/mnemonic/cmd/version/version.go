package version

import "fmt"

var (
	version   = "n/a"
	buildDate = "n/a"
	commit    = "n/a"
)

func Version() string {
	return version
}

func BuildDate() string {
	return buildDate
}

func Commit() string {
	return commit
}

func Print() string {
	const mnemonic = `
  __  __                                  _      
 |  \/  |                                (_)     
 | \  / |_ __   ___ _ __ ___   ___  _ __  _  ___ 
 | |\/| | '_ \ / _ \ '_ ' _ \ / _ \| '_ \| |/ __|
 | |  | | | | |  __/ | | | | | (_) | | | | | (__ 
 |_|  |_|_| |_|\___|_| |_| |_|\___/|_| |_|_|\___|`

	return fmt.Sprintf("%s\n                                   version %s\n", mnemonic, version)
}
