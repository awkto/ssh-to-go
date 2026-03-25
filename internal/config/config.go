package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Host struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
	User    string `yaml:"user"`
	KeyPath string `yaml:"key_path"`
}

type Config struct {
	ListenAddr   string        `yaml:"listen_addr"`
	PollInterval time.Duration `yaml:"poll_interval"`
	Hosts        []Host        `yaml:"hosts"`
}

// AppendHost adds a host to the config file by re-reading, appending, and re-writing.
func AppendHost(path string, host Host) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	// Get existing hosts list or create one
	hostsList, _ := raw["hosts"].([]interface{})
	newHost := map[string]interface{}{
		"name":    host.Name,
		"address": host.Address,
		"user":    host.User,
	}
	if host.KeyPath != "" {
		newHost["key_path"] = host.KeyPath
	}
	hostsList = append(hostsList, newHost)
	raw["hosts"] = hostsList

	out, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{
		ListenAddr:   "127.0.0.1:8080",
		PollInterval: 5 * time.Second,
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	for i := range cfg.Hosts {
		h := &cfg.Hosts[i]
		if h.KeyPath != "" && strings.HasPrefix(h.KeyPath, "~") {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("expand home dir: %w", err)
			}
			h.KeyPath = filepath.Join(home, h.KeyPath[1:])
		}
		if h.Address != "" && !strings.Contains(h.Address, ":") {
			h.Address = h.Address + ":22"
		}
	}

	if len(cfg.Hosts) == 0 {
		return nil, fmt.Errorf("no hosts configured")
	}

	return cfg, nil
}
