package version

// Set via -ldflags at build time.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func String() string {
	return Version + " (" + Commit + ") " + Date
}
