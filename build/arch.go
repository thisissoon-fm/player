package build

// Version string -ldflags "-X player/version.arch=x86_64"
var arch string

// Exported method for returning the architecture string
func Architecture() string {
	if arch == "" {
		return "n/a"
	}
	return arch
}
