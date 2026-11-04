package indexers

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Dates in definitions are parsed with .NET-style custom format strings
// ("dd MMM yyyy HH:mm zzz"). Instead of converting them to Go layouts, the format is
// matched token by token, which also lets month/day names be case-insensitive and
// numbers have one or two digits regardless of the token width.

// dateOut is the canonical text form filters hand to the next step.
const dateOut = time.RFC1123Z

// nowFunc is replaced in tests.
var nowFunc = func() time.Time { return time.Now().UTC() }

var monthNames = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}
var dayNames = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

type dateFields struct {
	year, month, day          int
	hour, min, sec, nsec      int
	pm                        int // -1 none, 0 am, 1 pm
	offset                    *int
	hasYear, hasMonth, hasDay bool
}

// parseDotNetDate parses value according to a .NET custom date format.
func parseDotNetDate(value, format string) (time.Time, error) {
	f := dateFields{pm: -1}
	v := value
	pos := 0
	fail := func() (time.Time, error) {
		return time.Time{}, fmt.Errorf("date %q does not match format %q", value, format)
	}
	for i := 0; i < len(format); {
		c := format[i]
		// literals
		if c == '\'' || c == '"' {
			j := strings.IndexByte(format[i+1:], c)
			if j < 0 {
				j = len(format) - i - 1
			}
			lit := format[i+1 : i+1+j]
			if !strings.HasPrefix(strings.ToLower(v[pos:]), strings.ToLower(lit)) {
				return fail()
			}
			pos += len(lit)
			i += j + 2
			continue
		}
		if c == '\\' && i+1 < len(format) {
			if pos >= len(v) || v[pos] != format[i+1] {
				return fail()
			}
			pos++
			i += 2
			continue
		}
		if c == '%' {
			i++
			continue
		}
		if c == ' ' || c == '\t' {
			for pos < len(v) && (v[pos] == ' ' || v[pos] == '\t' || v[pos] == '\n' || v[pos] == '\r' || v[pos] == 0xC2 && pos+1 < len(v) && v[pos+1] == 0xA0) {
				if v[pos] == 0xC2 {
					pos++
				}
				pos++
			}
			i++
			continue
		}
		if !strings.ContainsRune("dMyHhmsfFtzK", rune(c)) {
			// any other literal character must match (case-insensitive)
			r := []rune(format[i:])[0]
			if pos >= len(v) {
				return fail()
			}
			vr := []rune(v[pos:])[0]
			if unicode.ToLower(vr) != unicode.ToLower(r) {
				return fail()
			}
			pos += len(string(vr))
			i += len(string(r))
			continue
		}
		n := 1
		for i+n < len(format) && format[i+n] == c {
			n++
		}
		i += n
		var ok bool
		switch c {
		case 'd':
			if n <= 2 {
				f.day, pos, ok = readNum(v, pos, 1, 2)
				f.hasDay = true
			} else {
				_, pos, ok = readName(v, pos, dayNames)
			}
		case 'M':
			if n <= 2 {
				f.month, pos, ok = readNum(v, pos, 1, 2)
			} else {
				var m int
				m, pos, ok = readName(v, pos, monthNames)
				f.month = m + 1
			}
			f.hasMonth = true
		case 'y':
			var y int
			if n <= 2 {
				y, pos, ok = readNum(v, pos, 2, 2)
				if y < 50 {
					y += 2000
				} else {
					y += 1900
				}
			} else {
				y, pos, ok = readNum(v, pos, 4, 4)
			}
			f.year, f.hasYear = y, true
		case 'H', 'h':
			f.hour, pos, ok = readNum(v, pos, 1, 2)
		case 'm':
			f.min, pos, ok = readNum(v, pos, 1, 2)
		case 's':
			f.sec, pos, ok = readNum(v, pos, 1, 2)
		case 'f', 'F':
			start := pos
			for pos < len(v) && pos-start < n && v[pos] >= '0' && v[pos] <= '9' {
				pos++
			}
			ok = c == 'F' || pos-start == n
			if pos > start {
				frac, _ := strconv.Atoi((v[start:pos] + "000000000")[:9])
				f.nsec = frac
			}
		case 't':
			rest := strings.ToLower(v[pos:])
			switch {
			case strings.HasPrefix(rest, "am") || strings.HasPrefix(rest, "a.m."):
				f.pm, ok = 0, true
			case strings.HasPrefix(rest, "pm") || strings.HasPrefix(rest, "p.m."):
				f.pm, ok = 1, true
			case strings.HasPrefix(rest, "a"):
				f.pm, ok = 0, true
			case strings.HasPrefix(rest, "p"):
				f.pm, ok = 1, true
			}
			if ok {
				switch {
				case strings.HasPrefix(rest, "a.m.") || strings.HasPrefix(rest, "p.m."):
					pos += 4
				case n >= 2 && len(rest) >= 2 && (rest[1] == 'm'):
					pos += 2
				default:
					pos++
					if len(rest) >= 2 && rest[1] == 'm' {
						pos++
					}
				}
			}
		case 'z', 'K':
			var off int
			off, pos, ok = readOffset(v, pos)
			if ok {
				f.offset = &off
			}
		}
		if !ok {
			return fail()
		}
	}
	if strings.TrimSpace(v[pos:]) != "" {
		return fail()
	}
	return f.build()
}

func (f dateFields) build() (time.Time, error) {
	loc := time.UTC
	if f.offset != nil {
		loc = time.FixedZone("", *f.offset)
	}
	now := nowFunc().In(loc)
	if f.pm == 1 && f.hour < 12 {
		f.hour += 12
	} else if f.pm == 0 && f.hour == 12 {
		f.hour = 0
	}
	y, m, d := f.year, f.month, f.day
	if !f.hasYear {
		y = now.Year()
	}
	if !f.hasMonth {
		m = int(now.Month())
	}
	if !f.hasDay {
		d = now.Day()
	}
	if m < 1 || m > 12 || d < 1 || d > 31 || f.hour > 23 || f.min > 59 || f.sec > 60 {
		return time.Time{}, errors.New("date out of range")
	}
	t := time.Date(y, time.Month(m), d, f.hour, f.min, f.sec, f.nsec, loc)
	if !f.hasYear && t.After(now.Add(48*time.Hour)) {
		t = t.AddDate(-1, 0, 0) // a date without a year is in the past
	}
	return t, nil
}

func readNum(v string, pos, min, max int) (int, int, bool) {
	start := pos
	for pos < len(v) && pos-start < max && v[pos] >= '0' && v[pos] <= '9' {
		pos++
	}
	if pos-start < min {
		return 0, start, false
	}
	n, _ := strconv.Atoi(v[start:pos])
	return n, pos, true
}

// readName reads a run of letters and matches its first three against names.
func readName(v string, pos int, names []string) (int, int, bool) {
	start := pos
	for pos < len(v) {
		r := []rune(v[pos:])[0]
		if !unicode.IsLetter(r) {
			break
		}
		pos += len(string(r))
	}
	w := strings.ToLower(v[start:pos])
	if len(w) < 3 {
		return 0, start, false
	}
	for i, n := range names {
		if strings.HasPrefix(w, n) {
			return i, pos, true
		}
	}
	return 0, start, false
}

var tzAbbrev = map[string]int{
	"utc": 0, "gmt": 0, "z": 0, "wet": 0, "bst": 3600, "cet": 3600, "cest": 7200, "eet": 7200,
	"eest": 10800, "msk": 10800, "est": -5 * 3600, "edt": -4 * 3600, "cst": -6 * 3600,
	"cdt": -5 * 3600, "mst": -7 * 3600, "mdt": -6 * 3600, "pst": -8 * 3600, "pdt": -7 * 3600,
}

// readOffset reads "+03:00", "+0300", "+3", "-07", "Z", "UTC", "GMT+2".
func readOffset(v string, pos int) (int, int, bool) {
	start := pos
	j := pos
	for j < len(v) && unicode.IsLetter(rune(v[j])) {
		j++
	}
	base := 0
	if j > pos {
		off, ok := tzAbbrev[strings.ToLower(v[pos:j])]
		if !ok {
			return 0, start, false
		}
		base, pos = off, j
		if pos >= len(v) || (v[pos] != '+' && v[pos] != '-') {
			return base, pos, true
		}
	}
	if pos >= len(v) || (v[pos] != '+' && v[pos] != '-') {
		return 0, start, false
	}
	sign := 1
	if v[pos] == '-' {
		sign = -1
	}
	pos++
	h, p2, ok := readNum(v, pos, 1, 2)
	if !ok {
		return 0, start, false
	}
	pos = p2
	mins := 0
	if pos < len(v) && v[pos] == ':' {
		pos++
	}
	if m, p3, ok := readNum(v, pos, 2, 2); ok {
		mins, pos = m, p3
	}
	return base + sign*(h*3600+mins*60), pos, true
}

// ---- relative dates ----

var (
	agoPart   = regexp.MustCompile(`(\d+(?:[.,]\d+)?)\s*([a-z]+)`)
	articleRe = regexp.MustCompile(`\b(an?|one)\s+`)
	yearRe    = regexp.MustCompile(`\d{4}`)
	ordinalRe = regexp.MustCompile(`(\d)(st|nd|rd|th)\b`)
)

// parseTimeAgo handles "2 hours and 1 day ago", "9hr,12m,39s", "now", "yesterday".
func parseTimeAgo(s string) (time.Time, error) {
	now := nowFunc()
	l := strings.ToLower(strings.TrimSpace(s))
	switch l {
	case "now", "just now", "right now", "today":
		return now, nil
	case "yesterday":
		return now.Add(-24 * time.Hour), nil
	}
	l = articleRe.ReplaceAllString(l, "1 ")
	parts := agoPart.FindAllStringSubmatch(l, -1)
	if len(parts) == 0 {
		return time.Time{}, fmt.Errorf("not a relative time: %q", s)
	}
	t := now
	for _, p := range parts {
		n, _ := strconv.ParseFloat(strings.ReplaceAll(p[1], ",", "."), 64)
		u := p[2]
		var d time.Duration
		switch {
		case strings.HasPrefix(u, "mo") || u == "mon" || u == "mth":
			t = t.AddDate(0, -int(n), 0)
			continue
		case strings.HasPrefix(u, "s"):
			d = time.Second
		case strings.HasPrefix(u, "mi") || u == "m":
			d = time.Minute
		case strings.HasPrefix(u, "h"):
			d = time.Hour
		case strings.HasPrefix(u, "d"):
			d = 24 * time.Hour
		case strings.HasPrefix(u, "w"):
			d = 7 * 24 * time.Hour
		case strings.HasPrefix(u, "y"):
			t = t.AddDate(-int(n), 0, 0)
			continue
		default:
			return time.Time{}, fmt.Errorf("unknown time unit %q in %q", u, s)
		}
		t = t.Add(-time.Duration(n * float64(d)))
	}
	return t, nil
}

var (
	clockRe     = regexp.MustCompile(`(\d{1,2}):(\d{2})(?::(\d{2}))?\s*([ap]\.?m\.?)?`)
	unixRe      = regexp.MustCompile(`^\d{9,13}$`)
	relativeRe  = regexp.MustCompile(`^(\d+(?:[.,]\d+)?\s*[a-z]+[\s,]*(and\s+)?)+(ago)?$`)
	usFormats   = []string{"M/d/yyyy H:mm:ss", "M/d/yyyy H:mm", "M/d/yyyy h:mm tt", "M/d/yyyy", "M-d-yyyy H:mm", "M-d-yyyy", "M/d/yy", "M/d", "M-d"}
	ukFormats   = []string{"d/M/yyyy H:mm:ss", "d/M/yyyy H:mm", "d/M/yyyy h:mm tt", "d/M/yyyy", "d-M-yyyy H:mm", "d-M-yyyy", "d/M/yy", "d/M", "d-M"}
	baseFormats = []string{
		"yyyy-M-d H:mm:ss zzz", "yyyy-M-d H:mm:ss", "yyyy-M-d H:mm zzz", "yyyy-M-d H:mm", "yyyy-M-d",
		"yyyy-M-dTH:mm:ss", "yyyy/M/d H:mm:ss", "yyyy/M/d H:mm", "yyyy/M/d", "yyyy.M.d H:mm", "yyyy.M.d",
		"d.M.yyyy H:mm:ss", "d.M.yyyy H:mm", "d.M.yyyy", "d.M.yy H:mm", "d.M.yy",
		"ddd, d MMM yyyy H:mm:ss zzz", "ddd, d MMM yyyy H:mm:ss", "ddd d MMM yyyy H:mm",
		"d MMM yyyy H:mm:ss zzz", "d MMM yyyy H:mm:ss", "d MMM yyyy H:mm", "d MMM yyyy", "d MMM yy",
		"d MMM, yyyy", "d-MMM-yyyy H:mm", "d-MMM-yyyy", "d-MMM-yy", "MMM d yyyy H:mm", "MMM d yyyy",
		"MMM d, yyyy H:mm", "MMM d, yyyy h:mm tt", "MMM d, yyyy", "MMMM d, yyyy", "MMM d", "d MMM", "d MMM H:mm",
		"MMM d H:mm", "d MMM yyyy, H:mm", "H:mm d MMM yyyy", "yyyy MMM d",
	}
)

// parseFuzzy accepts most of what trackers print: relative times, today/yesterday,
// weekdays, UNIX timestamps and common absolute layouts. uk prefers day-first order.
func parseFuzzy(s string, uk bool) (time.Time, error) {
	raw := strings.TrimSpace(s)
	l := strings.ToLower(raw)
	if l == "" {
		return time.Time{}, errors.New("empty date")
	}
	now := nowFunc()
	if unixRe.MatchString(l) {
		n, _ := strconv.ParseInt(l, 10, 64)
		if len(l) > 11 {
			return time.UnixMilli(n).UTC(), nil
		}
		return time.Unix(n, 0).UTC(), nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC850, time.ANSIC} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	if strings.Contains(l, "ago") || relativeRe.MatchString(l) || l == "now" || l == "just now" {
		if t, err := parseTimeAgo(strings.ReplaceAll(l, "ago", "")); err == nil {
			return t, nil
		}
	}
	// today/yesterday/tomorrow/weekday [at] [time]
	var day time.Time
	switch {
	case strings.HasPrefix(l, "today"):
		day = now
	case strings.HasPrefix(l, "yesterday") || strings.HasPrefix(l, "y-day"):
		day = now.AddDate(0, 0, -1)
	case strings.HasPrefix(l, "tomorrow"):
		day = now.AddDate(0, 0, 1)
	default:
		for i, dn := range dayNames {
			if strings.HasPrefix(l, dn) && !strings.Contains(l, ",") && !yearRe.MatchString(l) {
				back := (int(now.Weekday()) - i + 7) % 7
				day = now.AddDate(0, 0, -back)
				break
			}
		}
	}
	if !day.IsZero() {
		y, m, d := day.Date()
		t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		if c := clockRe.FindStringSubmatch(l); c != nil {
			h, _ := strconv.Atoi(c[1])
			mi, _ := strconv.Atoi(c[2])
			sec, _ := strconv.Atoi(c[3])
			if strings.HasPrefix(c[4], "p") && h < 12 {
				h += 12
			} else if strings.HasPrefix(c[4], "a") && h == 12 {
				h = 0
			}
			t = t.Add(time.Duration(h)*time.Hour + time.Duration(mi)*time.Minute + time.Duration(sec)*time.Second)
		} else if day.Equal(now) {
			return now, nil
		}
		return t, nil
	}
	formats := append(append([]string{}, baseFormats...), usFormats...)
	if uk {
		formats = append(append([]string{}, baseFormats...), ukFormats...)
	}
	cleaned := strings.NewReplacer(" at ", " ", "  ", " ").Replace(raw)
	cleaned = ordinalRe.ReplaceAllString(cleaned, "$1")
	for _, f := range formats {
		if t, err := parseDotNetDate(cleaned, f); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised date %q", s)
}

// parseResultDate converts the final value of a "date" field.
func parseResultDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(dateOut, s); err == nil {
		return t.UTC(), true
	}
	if t, err := parseFuzzy(s, false); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}
