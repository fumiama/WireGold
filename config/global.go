package config

import "time"

// ShowDebugLog turn on to print verbose logs.
const ShowDebugLog = false

// Version will show in help message to distinguish different builds.
// Use -ldflags="-X github.com/fumiama/WireGold/config.Version=x.y.z" to override.
var Version = "dev"

func init() {
	if Version == "dev" {
		Version = "dev-" + time.Now().Format(time.DateOnly)
	}
}
