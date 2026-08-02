package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// cliConfig is the stored client configuration.
type cliConfig struct {
	URL       string `json:"url"`
	Token     string `json:"token,omitempty"`
	TokenName string `json:"token_name,omitempty"`
	// NoAuth records that the server was configured without authentication
	// at login time, so an empty Token is deliberate rather than broken.
	NoAuth bool `json:"no_auth,omitempty"`
	// New holds the remembered answers to `stogo new` prompts. Also
	// hand-editable — this file is the explicit control surface for those
	// defaults.
	New *newDefaults `json:"new,omitempty"`
}

// newDefaults are the `stogo new` prompt answers worth carrying to the next
// run. Dir and Command are stored as typed, $name/$date intact, so one
// remembered value stays correct for every future session name.
type newDefaults struct {
	Host string `json:"host,omitempty"`
	// Dir overrides the derived default (server new_session_dir + name
	// slug) when set. Never written by the prompts — the web form likewise
	// treats an edited directory as a one-off — but honored when set here.
	Dir string `json:"dir,omitempty"`
	// Command is nil when never answered (seed from the server's recent
	// commands), and points at "" when the remembered answer is "no
	// command".
	Command *string `json:"command,omitempty"`
	Attach  *bool   `json:"attach,omitempty"`
}

// updateNewDefaults mutates the stored `new` section and writes the file
// back. It re-reads the file rather than saving a loadConfig result, so
// STOGO_URL/STOGO_TOKEN env overrides never leak onto disk.
func updateNewDefaults(mut func(*newDefaults)) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	cfg := &cliConfig{}
	if data, rerr := os.ReadFile(path); rerr == nil {
		if jerr := json.Unmarshal(data, cfg); jerr != nil {
			return fmt.Errorf("parse %s: %w", path, jerr)
		}
	}
	if cfg.New == nil {
		cfg.New = &newDefaults{}
	}
	mut(cfg.New)
	return saveConfig(cfg)
}

func configPath() (string, error) {
	if p := os.Getenv("STOGO_CONFIG"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "stogo", "config.json"), nil
}

// loadConfig reads the stored config, applying STOGO_URL / STOGO_TOKEN
// overrides. Returns an error with login instructions when nothing is
// configured.
func loadConfig() (*cliConfig, error) {
	cfg := &cliConfig{}

	path, err := configPath()
	if err == nil {
		if data, rerr := os.ReadFile(path); rerr == nil {
			if jerr := json.Unmarshal(data, cfg); jerr != nil {
				return nil, fmt.Errorf("parse %s: %w", path, jerr)
			}
		}
	}

	if v := os.Getenv("STOGO_URL"); v != "" {
		cfg.URL = v
	}
	if v := os.Getenv("STOGO_TOKEN"); v != "" {
		cfg.Token = v
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("not logged in — run `stogo auth login` (or set STOGO_URL/STOGO_TOKEN)")
	}
	return cfg, nil
}

func saveConfig(cfg *cliConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}

func removeConfig() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
