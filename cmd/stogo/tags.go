package main

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"sort"
	"strings"

	"golang.org/x/term"
)

// Session tags: free-form labels ("work", "personal") shared with the web
// dashboard. They live in the server's per-session preferences
// (/api/session-icons) and each tag's colour in /api/tag-colors.

// sessionPrefs is one /api/session-icons entry. Raw keeps every field the
// server sent, including ones this client doesn't know about, so a tag edit
// writes back the icon, star, theme and the rest untouched.
type sessionPrefs map[string]any

func (c *apiClient) sessionPrefsAll() (map[string]sessionPrefs, error) {
	var out map[string]sessionPrefs
	err := c.do("GET", "/api/session-icons", nil, &out)
	return out, err
}

func (c *apiClient) tagColors() (map[string]int, error) {
	var out map[string]int
	err := c.do("GET", "/api/tag-colors", nil, &out)
	return out, err
}

func (p sessionPrefs) tags() []string {
	raw, _ := p["tags"].([]any)
	var out []string
	for _, t := range raw {
		if s, ok := t.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// normalizeTag mirrors the server's keystore.NormalizeTags for one tag.
func normalizeTag(t string) string {
	t = strings.Join(strings.Fields(strings.ToLower(t)), "-")
	if r := []rune(t); len(r) > 32 {
		t = string(r[:32])
	}
	return t
}

// resolveAnySession finds a live session first, then an offloaded one, so
// tags can be edited on either.
func (c *apiClient) resolveAnySession(arg string) (host, session string, err error) {
	host, session, err = c.resolveSession(arg)
	if err == nil {
		return host, session, nil
	}
	if h, off, oerr := c.resolveOffloaded(arg); oerr == nil {
		return h, off.Name, nil
	}
	return "", "", err
}

const tagUsage = `usage: stogo tag <session> [tag ...]     add tags (prefix one with - to remove it)
       stogo untag <session> <tag> ...   remove tags
       stogo tag <session>               show a session's tags`

func cmdTag(args []string, remove bool) error {
	if len(args) < 1 || (remove && len(args) < 2) {
		return fmt.Errorf("%s", tagUsage)
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	c := newClient(cfg)
	host, session, err := c.resolveAnySession(args[0])
	if err != nil {
		return err
	}
	all, err := c.sessionPrefsAll()
	if err != nil {
		return err
	}
	prefs := all[host+":"+session]
	if prefs == nil {
		prefs = sessionPrefs{}
	}
	colors, _ := c.tagColors()

	if len(args) == 1 {
		printTagLine(host, session, prefs.tags(), colors)
		return nil
	}

	set := map[string]bool{}
	for _, t := range prefs.tags() {
		set[t] = true
	}
	for _, a := range args[1:] {
		drop := remove
		if strings.HasPrefix(a, "-") {
			drop, a = true, a[1:]
		} else {
			a = strings.TrimPrefix(a, "+")
		}
		t := normalizeTag(a)
		if t == "" {
			continue
		}
		if drop {
			delete(set, t)
		} else {
			set[t] = true
		}
	}
	tags := make([]string, 0, len(set))
	for t := range set {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	prefs["tags"] = tags

	path := "/api/session-icons/" + url.PathEscape(host) + "/" + url.PathEscape(session)
	if err := c.do("PUT", path, prefs, nil); err != nil {
		return err
	}
	// New tags get their colour server-side on save; fetch it for display.
	colors, _ = c.tagColors()
	printTagLine(host, session, tags, colors)
	return nil
}

func printTagLine(host, session string, tags []string, colors map[string]int) {
	if len(tags) == 0 {
		fmt.Printf("%s/%s: no tags\n", host, session)
		return
	}
	fmt.Printf("%s/%s: %s\n", host, session, renderTags(tags, colors))
}

// cmdTags lists every tag in use with how many sessions carry it.
func cmdTags(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("usage: stogo tags")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	c := newClient(cfg)
	all, err := c.sessionPrefsAll()
	if err != nil {
		return err
	}
	colors, _ := c.tagColors()
	counts := map[string]int{}
	for _, p := range all {
		for _, t := range p.tags() {
			counts[t]++
		}
	}
	if len(counts) == 0 {
		fmt.Println("No tags yet — add one with: stogo tag <session> <tag>")
		return nil
	}
	names := make([]string, 0, len(counts))
	for t := range counts {
		names = append(names, t)
	}
	sort.Strings(names)
	for _, t := range names {
		fmt.Printf("%s  %d\n", renderTags([]string{t}, colors), counts[t])
	}
	return nil
}

// cmdCompleteTags feeds bash completion with every tag in use.
func cmdCompleteTags() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	all, err := newClient(cfg).sessionPrefsAll()
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, p := range all {
		for _, t := range p.tags() {
			if !seen[t] {
				seen[t] = true
				fmt.Println(t)
			}
		}
	}
	return nil
}

// renderTags joins tags with commas, each in its dashboard colour when
// stdout is a terminal that hasn't opted out via NO_COLOR.
func renderTags(tags []string, colors map[string]int) string {
	if len(tags) == 0 {
		return "-"
	}
	if !useColor() {
		return strings.Join(tags, ",")
	}
	parts := make([]string, len(tags))
	for i, t := range tags {
		r, g, b := tagRGB(t, colors)
		parts[i] = fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, t)
	}
	return strings.Join(parts, ",")
}

func useColor() bool {
	return os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(os.Stdout.Fd()))
}

// tagHue is the tag's stored hue, or the same name hash the web UI falls
// back to for tags created before colours were stored.
func tagHue(tag string, colors map[string]int) int {
	if h, ok := colors[tag]; ok {
		return h
	}
	var h uint32
	for _, c := range tag {
		h = h*31 + uint32(c)
	}
	return int(h % 360)
}

// tagRGB converts the tag's OKLCH colour (the dashboard's chip text colour,
// L 0.78 C 0.12) to 8-bit sRGB.
func tagRGB(tag string, colors map[string]int) (int, int, int) {
	hr := float64(tagHue(tag, colors)) * math.Pi / 180
	L, C := 0.78, 0.12
	a, b := C*math.Cos(hr), C*math.Sin(hr)
	l := math.Pow(L+0.3963377774*a+0.2158037573*b, 3)
	m := math.Pow(L-0.1055613458*a-0.0638541728*b, 3)
	s := math.Pow(L-0.0894841775*a-1.2914855480*b, 3)
	lin := [3]float64{
		4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		-0.0041960863*l - 0.7034186147*m + 1.7076147010*s,
	}
	var out [3]int
	for i, v := range lin {
		v = math.Max(0, math.Min(1, v))
		if v <= 0.0031308 {
			v *= 12.92
		} else {
			v = 1.055*math.Pow(v, 1/2.4) - 0.055
		}
		out[i] = int(math.Round(v * 255))
	}
	return out[0], out[1], out[2]
}
