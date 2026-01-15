package version

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
