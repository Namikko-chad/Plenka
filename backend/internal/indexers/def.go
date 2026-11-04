package indexers

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Definition is one Cardigann tracker definition as read from YAML. Only the keys the
// engine acts on are modelled; unknown keys are ignored.
type Definition struct {
	ID              string         `yaml:"id"`
	Replaces        []string       `yaml:"replaces"`
	Name            string         `yaml:"name"`
	Description     string         `yaml:"description"`
	Language        string         `yaml:"language"`
	Type            string         `yaml:"type"`
	Encoding        string         `yaml:"encoding"`
	FollowRedirect  bool           `yaml:"followredirect"`
	TestLinkTorrent *bool          `yaml:"testlinktorrent"`
	RequestDelay    float64        `yaml:"requestDelay"`
	Links           []string       `yaml:"links"`
	LegacyLinks     []string       `yaml:"legacylinks"`
	Certificates    []string       `yaml:"certificates"`
	Caps            Caps           `yaml:"caps"`
	Settings        *[]Setting     `yaml:"settings"`
	Login           *LoginBlock    `yaml:"login"`
	Search          SearchBlock    `yaml:"search"`
	Download        *DownloadBlock `yaml:"download"`
}

// Caps describes categories and search modes.
type Caps struct {
	CategoryMappings []CategoryMapping   `yaml:"categorymappings"`
	Categories       pairs               `yaml:"categories"` // older form: tracker id -> Newznab name
	Modes            map[string][]string `yaml:"modes"`
}

// CategoryMapping maps a tracker category id onto a Newznab category name.
type CategoryMapping struct {
	ID      string `yaml:"id"`
	Cat     string `yaml:"cat"`
	Desc    string `yaml:"desc"`
	Default bool   `yaml:"default"`
}

// Setting is one user-editable option of an indexer.
type Setting struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Label    string   `yaml:"label"`
	Default  string   `yaml:"default"`
	Defaults []string `yaml:"defaults"`
	Options  pairs    `yaml:"options"`
}

// LoginBlock describes how to obtain a session.
type LoginBlock struct {
	Path              string        `yaml:"path"`
	SubmitPath        string        `yaml:"submitpath"`
	Method            string        `yaml:"method"`
	Form              string        `yaml:"form"`
	Selectors         bool          `yaml:"selectors"`
	Inputs            pairs         `yaml:"inputs"`
	SelectorInputs    namedBlocks   `yaml:"selectorinputs"`
	GetSelectorInputs namedBlocks   `yaml:"getselectorinputs"`
	Cookies           []string      `yaml:"cookies"`
	Headers           headers       `yaml:"headers"`
	Error             []ErrorBlock  `yaml:"error"`
	Test              *PageTest     `yaml:"test"`
	Captcha           *CaptchaBlock `yaml:"captcha"`
}

// ErrorBlock: when Selector matches, the page reports an error (Message extracts it).
type ErrorBlock struct {
	Selector string         `yaml:"selector"`
	Message  *SelectorBlock `yaml:"message"`
}

// PageTest checks that a session is valid.
type PageTest struct {
	Path     string `yaml:"path"`
	Selector string `yaml:"selector"`
}

// CaptchaBlock marks a login form with a CAPTCHA.
type CaptchaBlock struct {
	Type     string `yaml:"type"`
	Selector string `yaml:"selector"`
	Input    string `yaml:"input"`
}

// SearchBlock describes the search requests and how to read the results.
type SearchBlock struct {
	Path                 string       `yaml:"path"`
	Paths                []SearchPath `yaml:"paths"`
	Inputs               pairs        `yaml:"inputs"`
	Headers              headers      `yaml:"headers"`
	KeywordsFilters      []FilterDef  `yaml:"keywordsfilters"`
	PreprocessingFilters []FilterDef  `yaml:"preprocessingfilters"`
	Error                []ErrorBlock `yaml:"error"`
	AllowEmptyInputs     bool         `yaml:"allowEmptyInputs"`
	Rows                 RowsBlock    `yaml:"rows"`
	Fields               namedBlocks  `yaml:"fields"`
}

// SearchPath is one request of a search.
type SearchPath struct {
	Path           string         `yaml:"path"`
	Method         string         `yaml:"method"`
	FollowRedirect bool           `yaml:"followredirect"`
	Categories     []string       `yaml:"categories"`
	Inputs         pairs          `yaml:"inputs"`
	InheritInputs  *bool          `yaml:"inheritinputs"`
	QuerySeparator string         `yaml:"queryseparator"`
	Response       *ResponseBlock `yaml:"response"`
}

// ResponseBlock selects JSON or XML parsing instead of HTML.
type ResponseBlock struct {
	Type             string  `yaml:"type"`
	NoResultsMessage *string `yaml:"noResultsMessage"`
}

// RowsBlock selects the result rows.
type RowsBlock struct {
	Selector                        string         `yaml:"selector"`
	Attribute                       string         `yaml:"attribute"`
	After                           int            `yaml:"after"`
	Filters                         []FilterDef    `yaml:"filters"`
	DateHeaders                     *SelectorBlock `yaml:"dateheaders"`
	Count                           *SelectorBlock `yaml:"count"`
	Multiple                        bool           `yaml:"multiple"`
	MissingAttributeEqualsNoResults bool           `yaml:"missingAttributeEqualsNoResults"`
}

// SelectorBlock extracts one value from a row or page.
type SelectorBlock struct {
	Selector          string      `yaml:"selector"`
	Attribute         string      `yaml:"attribute"`
	Text              *string     `yaml:"text"`
	Remove            string      `yaml:"remove"`
	Filters           []FilterDef `yaml:"filters"`
	Optional          bool        `yaml:"optional"`
	Default           *string     `yaml:"default"`
	Case              pairs       `yaml:"case"`
	UseBeforeResponse bool        `yaml:"usebeforeresponse"`
}

// FilterDef is one step of a filter chain.
type FilterDef struct {
	Name string   `yaml:"name"`
	Args argsList `yaml:"args"`
}

// DownloadBlock describes how to turn a result link into a torrent or magnet.
type DownloadBlock struct {
	Method    string          `yaml:"method"`
	Before    *BeforeBlock    `yaml:"before"`
	Selectors []SelectorBlock `yaml:"selectors"`
	Infohash  *InfohashBlock  `yaml:"infohash"`
	Headers   headers         `yaml:"headers"`
}

// BeforeBlock is a request made before the download.
type BeforeBlock struct {
	Path           string         `yaml:"path"`
	Method         string         `yaml:"method"`
	QuerySeparator string         `yaml:"queryseparator"`
	Inputs         pairs          `yaml:"inputs"`
	PathSelector   *SelectorBlock `yaml:"pathselector"`
}

// InfohashBlock builds a magnet from a page.
type InfohashBlock struct {
	Hash              SelectorBlock `yaml:"hash"`
	Title             SelectorBlock `yaml:"title"`
	UseBeforeResponse bool          `yaml:"usebeforeresponse"`
}

// ---- YAML helper types (order-preserving maps, scalar-or-list) ----

type pair struct{ Key, Value string }

// pairs is a YAML mapping of scalars, kept in document order.
type pairs []pair

func (p *pairs) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		return nil
	}
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: expected a mapping", n.Line)
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		val, err := scalarText(v)
		if err != nil {
			return err
		}
		*p = append(*p, pair{k.Value, val})
	}
	return nil
}

func (p pairs) get(key string) (string, bool) {
	for _, kv := range p {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return "", false
}

func scalarText(v *yaml.Node) (string, error) {
	switch v.Kind {
	case yaml.ScalarNode:
		if v.Tag == "!!null" {
			return "", nil
		}
		return v.Value, nil
	case yaml.AliasNode:
		if v.Alias != nil {
			return scalarText(v.Alias)
		}
	}
	return "", fmt.Errorf("line %d: expected a scalar value", v.Line)
}

// namedBlock is one entry of an ordered mapping name -> SelectorBlock (fields, selectorinputs).
type namedBlock struct {
	Name  string
	Block SelectorBlock
}

type namedBlocks []namedBlock

func (nb *namedBlocks) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		return nil
	}
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: expected a mapping", n.Line)
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		var b SelectorBlock
		if n.Content[i+1].Kind == yaml.ScalarNode {
			// shorthand "field: <text>" is not part of the format; treat as text
			t := n.Content[i+1].Value
			b.Text = &t
		} else if err := n.Content[i+1].Decode(&b); err != nil {
			return err
		}
		*nb = append(*nb, namedBlock{n.Content[i].Value, b})
	}
	return nil
}

// headers is an ordered mapping header -> one or more values.
type headers []struct {
	Name   string
	Values []string
}

func (h *headers) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		return nil
	}
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: expected a mapping", n.Line)
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		var vals argsList
		if err := vals.UnmarshalYAML(n.Content[i+1]); err != nil {
			return err
		}
		*h = append(*h, struct {
			Name   string
			Values []string
		}{n.Content[i].Value, vals})
	}
	return nil
}

// argsList accepts a scalar or a sequence of scalars.
type argsList []string

func (a *argsList) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		if n.Tag == "!!null" {
			return nil
		}
		*a = argsList{n.Value}
		return nil
	case yaml.SequenceNode:
		for _, c := range n.Content {
			s, err := scalarText(c)
			if err != nil {
				return err
			}
			*a = append(*a, s)
		}
		return nil
	}
	return fmt.Errorf("line %d: expected a scalar or a list", n.Line)
}

func (a argsList) arg(i int) string {
	if i < len(a) {
		return a[i]
	}
	return ""
}

// ---- loading ----

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// parseDefinition decodes YAML and applies defaults. It does not compile anything.
func parseDefinition(data []byte) (*Definition, error) {
	data = bytes.TrimPrefix(data, utf8BOM)
	var d Definition
	if err := yaml.Unmarshal(data, &d); err != nil {
		// YAML 1.2 allows the JSON escape "\/" in double-quoted scalars; the decoder
		// does not, so retry with it rewritten.
		fixed, changed := fixSlashEscapes(data)
		if !changed {
			return nil, err
		}
		d = Definition{}
		if err2 := yaml.Unmarshal(fixed, &d); err2 != nil {
			return nil, err
		}
	}
	d.ID = strings.TrimSpace(d.ID)
	if d.ID == "" {
		return nil, errors.New("missing id")
	}
	if d.Name == "" {
		d.Name = d.ID
	}
	if len(d.Links) == 0 {
		return nil, errors.New("no links")
	}
	for i, l := range d.Links {
		d.Links[i] = withSlash(strings.TrimSpace(l))
	}
	for i, l := range d.LegacyLinks {
		d.LegacyLinks[i] = withSlash(strings.TrimSpace(l))
	}
	d.Type = strings.ToLower(strings.TrimSpace(d.Type))
	if d.Encoding == "" {
		d.Encoding = "UTF-8"
	}
	for _, kv := range d.Caps.Categories { // legacy form
		d.Caps.CategoryMappings = append(d.Caps.CategoryMappings, CategoryMapping{ID: kv.Key, Cat: kv.Value})
	}
	if d.Search.Path != "" && len(d.Search.Paths) == 0 {
		d.Search.Paths = []SearchPath{{Path: d.Search.Path}}
	}
	if len(d.Search.Paths) == 0 {
		return nil, errors.New("search has no paths")
	}
	if len(d.Search.Fields) == 0 {
		return nil, errors.New("search has no fields")
	}
	return &d, nil
}

// settings returns the effective settings: a missing block means username + password.
func (d *Definition) settings() []Setting {
	if d.Settings == nil {
		return []Setting{
			{Name: "username", Type: "text", Label: "Username"},
			{Name: "password", Type: "password", Label: "Password"},
		}
	}
	return *d.Settings
}

// fixSlashEscapes rewrites `\/` (an odd run of backslashes before '/') to '/'.
func fixSlashEscapes(data []byte) ([]byte, bool) {
	out := make([]byte, 0, len(data))
	changed := false
	for i := 0; i < len(data); i++ {
		if data[i] != '\\' {
			out = append(out, data[i])
			continue
		}
		j := i
		for j < len(data) && data[j] == '\\' {
			j++
		}
		run := j - i
		if j < len(data) && data[j] == '/' && run%2 == 1 {
			out = append(out, data[i:j-1]...)
			changed = true
		} else {
			out = append(out, data[i:j]...)
		}
		i = j - 1
	}
	return out, changed
}

func withSlash(s string) string {
	if s != "" && !strings.HasSuffix(s, "/") {
		return s + "/"
	}
	return s
}
