package wingetrun

const (
	WINGET_API_ENDPOINT = "https://api.winget.run"
	WINGET_API_SEARCH   = "v2/packages"
)

type PackagesResult struct {
	Packages []Package `json:"Packages"`
	Total    int       `json:"Total"`
}

type Package struct {
	CreatedAt string `json:"CreatedAt"`
	Featured  bool   `json:"Featured"`
	ID        string `json:"Id"`
	Latest    struct {
		Description string   `json:"Description"`
		Homepage    string   `json:"Homepage"`
		License     string   `json:"License"`
		LicenseURL  *string  `json:"LicenseUrl"`
		Name        string   `json:"Name"`
		Publisher   string   `json:"Publisher"`
		Tags        []string `json:"Tags"`
	} `json:"Latest"`
	Search struct {
		Description string `json:"Description"`
		Name        string `json:"Name"`
		Publisher   string `json:"Publisher"`
		Tags        string `json:"Tags"`
	} `json:"Search"`
	SearchScore int      `json:"SearchScore"`
	Versions    []string `json:"Versions"`
}

type PackageSearchFilter struct {
	query          string
	name           string
	publisher      string
	description    string
	tags           string
	splitQuery     bool
	partialMatch   bool
	ensureContains bool
	preferContains bool
	sample         int
}

func PackageSearch(search PackageSearchFilter) PackagesResult {

}
