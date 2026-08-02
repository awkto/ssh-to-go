package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// apiClient is a thin HTTP wrapper around the ssh-to-go API using bearer
// token auth.
type apiClient struct {
	base  string
	token string
	http  *http.Client
}

func newClient(cfg *cliConfig) *apiClient {
	return &apiClient{
		base:  strings.TrimRight(cfg.URL, "/"),
		token: cfg.Token,
		http:  &http.Client{Timeout: 30 * time.Second},
	}
}

// do performs a request and decodes the JSON response into out (unless nil).
func (c *apiClient) do(method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.base+path, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return apiError(resp.StatusCode, data)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}
	}
	return nil
}

func apiError(status int, body []byte) error {
	// Handlers return either {"error": "..."} or plain-text http.Error output.
	var e struct {
		Error string `json:"error"`
	}
	msg := strings.TrimSpace(string(body))
	if json.Unmarshal(body, &e) == nil && e.Error != "" {
		msg = e.Error
	}
	if status == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized — token may have been revoked, run `stogo auth login` again")
	}
	if msg == "" {
		msg = http.StatusText(status)
	}
	return fmt.Errorf("%s (HTTP %d)", msg, status)
}

// API response shapes (mirrors internal/hub and internal/tmux wire formats).

type tmuxSession struct {
	// ID is the server-assigned short numeric ID (unique across hosts).
	// Zero when talking to a server predating the feature.
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Windows         int       `json:"windows"`
	Created         time.Time `json:"created"`
	Activity        time.Time `json:"activity"`
	Attached        bool      `json:"attached"`
	AttachedClients int       `json:"attached_clients"`
}

type hostSession struct {
	HostName string      `json:"host_name"`
	Session  tmuxSession `json:"session"`
}

type hostState struct {
	Config struct {
		Name string `json:"name"`
	} `json:"config"`
	Online   bool          `json:"online"`
	Error    string        `json:"error,omitempty"`
	Sessions []tmuxSession `json:"sessions"`
}

// serverSettings is the subset of GET /api/settings the CLI cares about.
// An old server that lacks a field just yields the zero value, which every
// caller treats as "no default configured".
type serverSettings struct {
	DefaultHost   string `json:"default_host"`
	NewSessionDir string `json:"new_session_dir"`
}

type recentCommand struct {
	Command string `json:"command"`
}

// createSessionReq mirrors the server's create-session request body. Cwd and
// Command may contain $name/$date — the server expands them after
// sanitizing the name, so the CLI never has to.
type createSessionReq struct {
	Name      string `json:"name"`
	Cwd       string `json:"cwd,omitempty"`
	CreateDir bool   `json:"create_dir,omitempty"`
	Command   string `json:"command,omitempty"`
}

func (c *apiClient) sessions() ([]hostSession, error) {
	var out []hostSession
	err := c.do("GET", "/api/sessions", nil, &out)
	return out, err
}

func (c *apiClient) hosts() ([]hostState, error) {
	var out []hostState
	err := c.do("GET", "/api/hosts", nil, &out)
	return out, err
}

func (c *apiClient) settings() (serverSettings, error) {
	var out serverSettings
	err := c.do("GET", "/api/settings", nil, &out)
	return out, err
}

func (c *apiClient) recentCommands() ([]recentCommand, error) {
	var out []recentCommand
	err := c.do("GET", "/api/recent-commands", nil, &out)
	return out, err
}

func (c *apiClient) createSession(host string, req createSessionReq) error {
	return c.do("POST", "/api/hosts/"+url.PathEscape(host)+"/sessions", req, nil)
}

// resolveSession turns NAME, HOST/NAME or a numeric ID (as shown by
// `stogo list`) into a concrete (host, session) pair, rescanning once if
// nothing matches. An exact name match wins over an ID interpretation, so a
// session literally named "3" is still addressable.
func (c *apiClient) resolveSession(arg string) (host, session string, err error) {
	if h, s, ok := strings.Cut(arg, "/"); ok {
		return h, s, nil
	}

	find := func() ([]hostSession, error) {
		all, err := c.sessions()
		if err != nil {
			return nil, err
		}
		var matches []hostSession
		for _, hs := range all {
			if hs.Session.Name == arg {
				matches = append(matches, hs)
			}
		}
		if len(matches) == 0 {
			if id, aerr := strconv.Atoi(arg); aerr == nil && id > 0 {
				for _, hs := range all {
					if hs.Session.ID == id {
						matches = append(matches, hs)
					}
				}
			}
		}
		return matches, nil
	}

	matches, err := find()
	if err != nil {
		return "", "", err
	}
	if len(matches) == 0 {
		// The dashboard polls hosts on an interval; a session created
		// seconds ago may not be listed yet. One forced scan closes that gap.
		if scanErr := c.do("POST", "/api/scan", nil, nil); scanErr == nil {
			matches, err = find()
			if err != nil {
				return "", "", err
			}
		}
	}
	switch len(matches) {
	case 0:
		return "", "", fmt.Errorf("no session with name or ID %q found (try `stogo list`)", arg)
	case 1:
		return matches[0].HostName, matches[0].Session.Name, nil
	default:
		var hosts []string
		for _, m := range matches {
			hosts = append(hosts, m.HostName+"/"+m.Session.Name)
		}
		return "", "", fmt.Errorf("session %q exists on multiple hosts — use one of: %s",
			arg, strings.Join(hosts, ", "))
	}
}
