package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Host struct {
	Name    string `yaml:"name"     json:"name"`
	Address string `yaml:"address"  json:"address"`
	Port    int    `yaml:"port"     json:"port"`
	User    string `yaml:"user"     json:"user"`
	KeyPath string `yaml:"key_path" json:"key_path,omitempty"`
	KeyName string `yaml:"key_name" json:"key_name,omitempty"`
	OS      string `yaml:"os"       json:"os,omitempty"`
}

// DialAddress returns host:port for SSH dialing.
func (h Host) DialAddress() string {
	port := h.Port
	if port == 0 {
		port = 22
	}
	return fmt.Sprintf("%s:%d", h.Address, port)
}

type Config struct {
	ListenAddr   string        `yaml:"listen_addr"`
	PollInterval time.Duration `yaml:"poll_interval"`
	DataDir      string        `yaml:"data_dir"`
	Hosts        []Host        `yaml:"hosts"`
}

// AppendHost adds a host to the config file by re-reading, appending, and re-writing.
// Creates the config file if it doesn't exist.
func AppendHost(path string, host Host) error {
	var raw map[string]interface{}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			raw = map[string]interface{}{}
		} else {
			return fmt.Errorf("read config: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parse config: %w", err)
		}
	}
	if raw == nil {
		raw = map[string]interface{}{}
	}

	// Get existing hosts list or create one
	hostsList, _ := raw["hosts"].([]interface{})
	newHost := map[string]interface{}{
		"name":    host.Name,
		"address": host.Address,
		"user":    host.User,
	}
	if host.Port != 0 && host.Port != 22 {
		newHost["port"] = host.Port
	}
	if host.KeyPath != "" {
		newHost["key_path"] = host.KeyPath
	}
	if host.KeyName != "" {
		newHost["key_name"] = host.KeyName
	}
	if host.OS != "" {
		newHost["os"] = host.OS
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

// UpdateHost updates a host in the config file by name.
func UpdateHost(path string, name string, updated Host) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	hostsList, _ := raw["hosts"].([]interface{})
	found := false
	for i, entry := range hostsList {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if m["name"] == name {
			newHost := map[string]interface{}{
				"name":    updated.Name,
				"address": updated.Address,
				"user":    updated.User,
			}
			if updated.Port != 0 && updated.Port != 22 {
				newHost["port"] = updated.Port
			}
			if updated.KeyName != "" {
				newHost["key_name"] = updated.KeyName
			}
			if updated.KeyPath != "" {
				newHost["key_path"] = updated.KeyPath
			}
			if updated.OS != "" {
				newHost["os"] = updated.OS
			}
			hostsList[i] = newHost
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("host %q not found in config", name)
	}

	raw["hosts"] = hostsList
	out, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, out, 0644)
}

// RemoveHost removes a host from the config file by name.
func RemoveHost(path string, name string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	hostsList, _ := raw["hosts"].([]interface{})
	found := false
	for i, entry := range hostsList {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if m["name"] == name {
			hostsList = append(hostsList[:i], hostsList[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("host %q not found in config", name)
	}

	raw["hosts"] = hostsList
	out, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, out, 0644)
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		ListenAddr:   "127.0.0.1:8080",
		PollInterval: 5 * time.Second,
	}

	absPath, _ := filepath.Abs(path)
	configDir := filepath.Dir(absPath)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file — start with defaults and no hosts
			cfg.DataDir = filepath.Join(configDir, "data")
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
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
		// Backwards compat: if address has an embedded port, extract it
		if h.Address != "" && strings.Contains(h.Address, ":") {
			parts := strings.SplitN(h.Address, ":", 2)
			h.Address = parts[0]
			if h.Port == 0 {
				p, _ := strconv.Atoi(parts[1])
				if p > 0 {
					h.Port = p
				}
			}
		}
		if h.Port == 0 {
			h.Port = 22
		}
	}

	if cfg.DataDir == "" {
		cfg.DataDir = "data"
	}
	if strings.HasPrefix(cfg.DataDir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("expand home dir: %w", err)
		}
		cfg.DataDir = filepath.Join(home, cfg.DataDir[1:])
	} else if !filepath.IsAbs(cfg.DataDir) {
		cfg.DataDir = filepath.Join(configDir, cfg.DataDir)
	}

	return cfg, nil
}
