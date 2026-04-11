package config

import "time"

// ShowDebugLog turn on to print verbose logs.
const ShowDebugLog = false

// Version will show in help message to distinguish different builds.
var Version = "dev-" + time.Now().Format(time.DateOnly)
