package keystore

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// iconPalette and colorPalette mirror the icon/color names defined in
// web/static/js/icons.js. They exist so the server can assign a random
// icon+color to a newly created session (see Settings.SessionIconMode).
// Keep in sync when icons/colors are added or removed on the web side.
var iconPalette = []string{
	"terminal", "code", "git-commit", "git-branch", "cloud", "hash", "at-sign",
	"heart", "star", "home", "bookmark", "flag", "tag", "user", "users",
	"briefcase", "calendar", "clock", "map-pin", "compass", "sun", "moon",
	"coffee", "anchor", "truck", "package", "crosshair", "award", "feather",
	"hexagon", "droplet", "scissors", "paperclip", "server", "database",
	"monitor", "laptop", "smartphone", "tablet", "cpu", "hard-drive", "wifi",
	"globe", "lock", "unlock", "shield", "key", "folder", "file", "file-text",
	"bug", "zap", "activity", "layers", "box", "settings", "tool", "command",
	"link", "eye", "bell", "send", "download", "upload", "refresh", "power",
	"bluetooth", "radio", "router", "gamepad", "music", "camera",
}

var colorPalette = []string{
	"default", "green", "red", "orange", "purple", "pink", "cyan", "gray",
	"amber", "yellow", "lime", "emerald", "rose", "crimson", "coral", "bronze",
	"teal", "sky", "indigo", "violet", "fuchsia", "magenta", "midnight",
	"forest", "mint", "peach", "lavender", "blush", "sand", "slate",
	"charcoal", "white",
}

// RandomSessionIcon returns a SessionIcon with a randomly chosen icon and
// color from the built-in palettes. Go's global rand is auto-seeded, so
// successive sessions get an unpredictable spread.
func RandomSessionIcon() SessionIcon {
	return SessionIcon{
		Icon:  iconPalette[rand.Intn(len(iconPalette))],
		Color: colorPalette[rand.Intn(len(colorPalette))],
	}
}

type SessionIcon struct {
	Icon         string `json:"icon,omitempty"`
	Color        string `json:"color,omitempty"`
	Starred      bool   `json:"starred,omitempty"`
	Theme        string `json:"theme,omitempty"`
	LastAccessed string `json:"last_accessed,omitempty"`
	// KeepAwake exempts the session from idle auto-offload. It lives here
	// with the other per-session user preferences (star, theme) rather than
	// in the registry, which records what a session IS rather than how its
	// owner wants it treated.
	KeepAwake bool `json:"keep_awake,omitempty"`
	// Tags are free-form labels ("work", "personal") the sessions page
	// filters and groups by. Stored normalised; see NormalizeTags.
	Tags []string `json:"tags,omitempty"`
}

// Tag limits. Generous for labels, small enough that a hostile client
// can't bloat session-icons.json.
const (
	maxTags   = 16
	maxTagLen = 32
)

// NormalizeTags lowercases and trims each tag, turns inner whitespace into
// dashes, drops empties and duplicates, and returns the rest sorted, so
// "Work", " work " and "work" are one tag. Tags past the length or count
// limits are cut. Returns nil for no tags so the field stays omitted.
func NormalizeTags(in []string) []string {
	seen := make(map[string]bool, len(in))
	var out []string
	for _, t := range in {
		t = strings.Join(strings.Fields(strings.ToLower(t)), "-")
		if r := []rune(t); len(r) > maxTagLen {
			t = string(r[:maxTagLen])
		}
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	sort.Strings(out)
	if len(out) > maxTags {
		out = out[:maxTags]
	}
	return out
}

// TagHues is the swatch palette tag colours are drawn from: OKLCH hues
// (red, orange, amber, yellow, lime, green, teal, cyan, blue, indigo,
// violet, pink). Mirrors TAG_HUES in web/static/app/sessions.jsx.
var TagHues = []int{25, 55, 75, 95, 125, 150, 180, 210, 245, 275, 305, 345}

// SessionIconStore persists session icon/color overrides to a JSON file,
// plus the colour of each tag, which is shared by every session carrying it.
type SessionIconStore struct {
	mu       sync.RWMutex
	path     string
	icons    map[string]SessionIcon // key: "host:session"
	tagsPath string
	tagHues  map[string]int // tag -> OKLCH hue, 0-359
}

func NewSessionIconStore(dataDir string) (*SessionIconStore, error) {
	s := &SessionIconStore{
		path:     filepath.Join(dataDir, "session-icons.json"),
		icons:    make(map[string]SessionIcon),
		tagsPath: filepath.Join(dataDir, "tag-colors.json"),
		tagHues:  make(map[string]int),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	if data, err := os.ReadFile(s.tagsPath); err == nil {
		if err := json.Unmarshal(data, &s.tagHues); err != nil {
			return nil, fmt.Errorf("read tag colors: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read tag colors: %w", err)
	}
	return s, nil
}

func (s *SessionIconStore) saveTagHues() error {
	data, err := json.MarshalIndent(s.tagHues, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.tagsPath, data, 0600)
}

// TagColors returns every tag's hue. A tag keeps its colour after the last
// session drops it, so re-adding it brings the same colour back.
func (s *SessionIconStore) TagColors() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]int, len(s.tagHues))
	for k, v := range s.tagHues {
		cp[k] = v
	}
	return cp
}

// SetTagColor sets one tag's hue (wrapped into 0-359).
func (s *SessionIconStore) SetTagColor(tag string, hue int) error {
	norm := NormalizeTags([]string{tag})
	if len(norm) == 0 {
		return fmt.Errorf("empty tag")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tagHues[norm[0]] = ((hue % 360) + 360) % 360
	return s.saveTagHues()
}

func (s *SessionIconStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read session icons: %w", err)
	}
	return json.Unmarshal(data, &s.icons)
}

func (s *SessionIconStore) save() error {
	data, err := json.MarshalIndent(s.icons, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func sessionKey(host, session string) string {
	return host + ":" + session
}

func (s *SessionIconStore) Get(host, session string) SessionIcon {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.icons[sessionKey(host, session)]
}

func (s *SessionIconStore) GetAll() map[string]SessionIcon {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]SessionIcon, len(s.icons))
	for k, v := range s.icons {
		cp[k] = v
	}
	return cp
}

func (s *SessionIconStore) Set(host, session string, icon SessionIcon) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sessionKey(host, session)
	icon.Tags = NormalizeTags(icon.Tags)
	// A tag seen for the first time gets a random colour from the palette.
	fresh := false
	for _, t := range icon.Tags {
		if _, ok := s.tagHues[t]; !ok {
			s.tagHues[t] = TagHues[rand.Intn(len(TagHues))]
			fresh = true
		}
	}
	if fresh {
		if err := s.saveTagHues(); err != nil {
			return err
		}
	}
	if icon.Icon == "" && icon.Color == "" && icon.Theme == "" && icon.LastAccessed == "" && !icon.Starred && !icon.KeepAwake && len(icon.Tags) == 0 {
		delete(s.icons, key)
	} else {
		s.icons[key] = icon
	}
	return s.save()
}

// Delete drops all icon/color/star/theme data for a session. Used when a
// session is collected and must leave no trace behind. Missing keys are not
// an error.
func (s *SessionIconStore) Delete(host, session string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sessionKey(host, session)
	if _, ok := s.icons[key]; !ok {
		return nil
	}
	delete(s.icons, key)
	return s.save()
}

// Rename migrates icon data from one session name to another.
func (s *SessionIconStore) Rename(host, oldSession, newSession string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldKey := sessionKey(host, oldSession)
	entry, ok := s.icons[oldKey]
	if !ok {
		return nil
	}
	delete(s.icons, oldKey)
	s.icons[sessionKey(host, newSession)] = entry
	return s.save()
}

// Touch updates the last-accessed timestamp for a session.
func (s *SessionIconStore) Touch(host, session string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sessionKey(host, session)
	entry := s.icons[key]
	entry.LastAccessed = time.Now().UTC().Format(time.RFC3339)
	s.icons[key] = entry
	return s.save()
}
