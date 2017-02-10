package build

// Version string -ldflags "-X player/version.os=darwin"
var os string

// Exported method for returning the os string
func OS() string {
	if os == "" {
		return "n/a"
	}
	return os
}
