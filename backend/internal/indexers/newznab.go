package indexers

import (
	"sort"
	"strings"
)

// Standard Newznab/Torznab categories (public specification).
var newznabCats = []struct {
	ID   int
	Name string
}{
	{1000, "Console"}, {1010, "Console/NDS"}, {1020, "Console/PSP"}, {1030, "Console/Wii"},
	{1040, "Console/XBox"}, {1050, "Console/XBox 360"}, {1060, "Console/Wiiware"},
	{1070, "Console/XBox 360 DLC"}, {1080, "Console/PS3"}, {1090, "Console/Other"},
	{1110, "Console/3DS"}, {1120, "Console/PS Vita"}, {1130, "Console/WiiU"},
	{1140, "Console/XBox One"}, {1180, "Console/PS4"},
	{2000, "Movies"}, {2010, "Movies/Foreign"}, {2020, "Movies/Other"}, {2030, "Movies/SD"},
	{2040, "Movies/HD"}, {2045, "Movies/UHD"}, {2050, "Movies/BluRay"}, {2060, "Movies/3D"},
	{2070, "Movies/DVD"}, {2080, "Movies/WEB-DL"}, {2090, "Movies/x265"},
	{3000, "Audio"}, {3010, "Audio/MP3"}, {3020, "Audio/Video"}, {3030, "Audio/Audiobook"},
	{3040, "Audio/Lossless"}, {3050, "Audio/Other"}, {3060, "Audio/Foreign"},
	{4000, "PC"}, {4010, "PC/0day"}, {4020, "PC/ISO"}, {4030, "PC/Mac"},
	{4040, "PC/Mobile-Other"}, {4050, "PC/Games"}, {4060, "PC/Mobile-iOS"},
	{4070, "PC/Mobile-Android"},
	{5000, "TV"}, {5010, "TV/WEB-DL"}, {5020, "TV/Foreign"}, {5030, "TV/SD"}, {5040, "TV/HD"},
	{5045, "TV/UHD"}, {5050, "TV/Other"}, {5060, "TV/Sport"}, {5070, "TV/Anime"},
	{5080, "TV/Documentary"}, {5090, "TV/x265"},
	{6000, "XXX"}, {6010, "XXX/DVD"}, {6020, "XXX/WMV"}, {6030, "XXX/XviD"}, {6040, "XXX/x264"},
	{6045, "XXX/UHD"}, {6050, "XXX/Pack"}, {6060, "XXX/ImageSet"}, {6070, "XXX/Other"},
	{6080, "XXX/SD"}, {6090, "XXX/WEB-DL"},
	{7000, "Books"}, {7010, "Books/Mags"}, {7020, "Books/EBook"}, {7030, "Books/Comics"},
	{7040, "Books/Technical"}, {7050, "Books/Other"}, {7060, "Books/Foreign"},
	{8000, "Other"}, {8010, "Other/Misc"}, {8020, "Other/Hashed"},
}

var (
	newznabByName = map[string]int{}
	newznabByID   = map[int]string{}
)

func init() {
	for _, c := range newznabCats {
		newznabByName[strings.ToLower(c.Name)] = c.ID
		newznabByID[c.ID] = c.Name
	}
}

// newznabID resolves a category name ("Movies/HD") or a numeric id ("2040").
func newznabID(name string) (int, bool) {
	name = strings.TrimSpace(name)
	if id, ok := newznabByName[strings.ToLower(name)]; ok {
		return id, true
	}
	if n, ok := atoiStrict(name); ok {
		if _, known := newznabByID[n]; known {
			return n, true
		}
	}
	return 0, false
}

func parentCat(id int) int { return id - id%1000 }

// catMap translates between tracker category ids and Newznab ids for one indexer.
type catMap struct {
	maps     []CategoryMapping
	nz       []int            // Newznab id per mapping (0 = unknown name)
	byID     map[string][]int // tracker id -> Newznab ids
	byDesc   map[string][]int // lower-cased tracker description -> Newznab ids
	defaults []string         // tracker ids marked default
}

func newCatMap(c Caps) *catMap {
	cm := &catMap{byID: map[string][]int{}, byDesc: map[string][]int{}}
	for _, m := range c.CategoryMappings {
		id, _ := newznabID(m.Cat)
		cm.maps = append(cm.maps, m)
		cm.nz = append(cm.nz, id)
		if id == 0 {
			continue
		}
		cm.byID[m.ID] = unionInts(cm.byID[m.ID], []int{id})
		if m.Desc != "" {
			k := strings.ToLower(strings.TrimSpace(m.Desc))
			cm.byDesc[k] = unionInts(cm.byDesc[k], []int{id})
		}
		if m.Default {
			cm.defaults = append(cm.defaults, m.ID)
		}
	}
	return cm
}

// catMatches reports whether Newznab category have satisfies a requested one
// (a request for a parent category matches every sub-category).
func catMatches(have, want int) bool {
	return have == want || (want%1000 == 0 && parentCat(have) == want)
}

// toTracker maps requested Newznab ids to tracker ids (defaults when none requested).
func (cm *catMap) toTracker(want []int) []string {
	if len(want) == 0 {
		return append([]string(nil), cm.defaults...)
	}
	var out []string
	seen := map[string]bool{}
	for i, m := range cm.maps {
		if seen[m.ID] || cm.nz[i] == 0 {
			continue
		}
		for _, w := range want {
			if catMatches(cm.nz[i], w) {
				seen[m.ID] = true
				out = append(out, m.ID)
				break
			}
		}
	}
	return out
}

// supports reports whether any mapping satisfies the request (empty request: yes).
func (cm *catMap) supports(want []int) bool {
	if len(want) == 0 {
		return true
	}
	for _, id := range cm.nz {
		for _, w := range want {
			if id != 0 && catMatches(id, w) {
				return true
			}
		}
	}
	return false
}

// fromTracker maps a tracker category id to Newznab ids (with parents added).
func (cm *catMap) fromTracker(id string) []int {
	return withParents(cm.byID[strings.TrimSpace(id)])
}

// fromDesc maps a tracker category description to Newznab ids.
func (cm *catMap) fromDesc(desc string) []int {
	return withParents(cm.byDesc[strings.ToLower(strings.TrimSpace(desc))])
}

// newznab lists the distinct Newznab categories of the mappings, sorted.
func (cm *catMap) newznab() []int {
	var out []int
	for _, id := range cm.nz {
		if id != 0 {
			out = unionInts(out, []int{id})
		}
	}
	sort.Ints(out)
	return out
}

func withParents(ids []int) []int {
	var out []int
	for _, id := range ids {
		if p := parentCat(id); p != id {
			out = unionInts(out, []int{p})
		}
		out = unionInts(out, []int{id})
	}
	return out
}

func unionInts(a, b []int) []int {
	for _, x := range b {
		found := false
		for _, y := range a {
			if x == y {
				found = true
				break
			}
		}
		if !found {
			a = append(a, x)
		}
	}
	return a
}

func atoiStrict(s string) (int, bool) {
	if s == "" || len(s) > 9 {
		return 0, false
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		n = n*10 + int(s[i]-'0')
	}
	return n, true
}
