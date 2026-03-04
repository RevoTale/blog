package metagen

type Metadata struct {
	Title       string
	Description string
	Alternates  Alternates
	Robots      *Robots
	OpenGraph   *OpenGraph
	Twitter     *Twitter
	Authors     []Author
	Publisher   string
	Pinterest   *Pinterest
	// DangerRawHead contains trusted raw HTML snippets rendered into <head>.
	// Never populate this field from third-party or user-controlled input.
	DangerRawHead []string
}

type Alternates struct {
	Canonical string
	Languages map[string]string
	Types     map[string]string
}

type Robots struct {
	Index  *bool
	Follow *bool
}

type OpenGraph struct {
	Type          string
	URL           string
	SiteName      string
	Title         string
	Description   string
	Locale        string
	PublishedTime string
	Authors       []string
	Tags          []string
	Images        []OpenGraphImage
}

type OpenGraphImage struct {
	URL    string
	Alt    string
	Width  int
	Height int
}

type Twitter struct {
	Card        string
	Site        string
	Creator     string
	Title       string
	Description string
	Images      []string
}

type Author struct {
	Name string
	URL  string
}

type Pinterest struct {
	RichPin *bool
}

type Patch struct {
	Title string `json:"title,omitempty"`
	Head  string `json:"head,omitempty"`
}

const HTMXPatchEvent = "metagen:patch"

func Bool(value bool) *bool {
	copy := value
	return &copy
}
