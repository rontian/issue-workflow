package version

type Info struct {
	Version   string `json:"version"`
	Protocol  string `json:"protocol"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

var (
	Version   = "dev"
	Protocol  = "v1"
	Commit    = "unknown"
	BuildDate = "unknown"
)
