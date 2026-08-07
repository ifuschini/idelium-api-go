package buildinfo

// Values are injected by release builds through linker flags.
var (
	Version = "dev"
	Commit  = "unknown"
)

// Info identifies the running API build.
type Info struct {
	Service string `json:"service"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// Current returns safe build metadata for diagnostics.
func Current() Info {
	return Info{
		Service: "idelium-api-go",
		Version: Version,
		Commit:  Commit,
	}
}
