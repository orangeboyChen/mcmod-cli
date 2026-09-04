// File: internal/config/config.go
// Created: 2026-06-20
// Description: Configuration management for project API keys.

package config

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Store stores the current project configuration for mcmod.
type Store struct {
	CFKey string `json:"cfKey,omitempty"`
}

// ReadEnvCFKey returns the CurseForge API key from environment.
func ReadEnvCFKey() string {
	return os.Getenv("CURSEFORGE_API_KEY")
}

const projectConfigDir = ".mcmod"

func projectConfigPath() string { return filepath.Join(projectConfigDir, "config.json") }

// ReadUserConfig is a compatibility alias for ReadProjectConfig.
func ReadUserConfig() (*Store, error) {
	return ReadProjectConfig()
}

// WriteUserConfig is a compatibility alias for WriteProjectConfig.
func WriteUserConfig(key string) error {
	return WriteProjectConfig(key)
}

// ReadProjectConfig returns project config from .mcmod/config.json.
func ReadProjectConfig() (*Store, error) {
	data, err := os.ReadFile(projectConfigPath())
	if err != nil {
		return nil, nil
	}
	var cfg Store
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// WriteProjectConfig writes project config to .mcmod/config.json.
func WriteProjectConfig(key string) error {
	dir := projectConfigDir
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	cfg := &Store{CFKey: key}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(projectConfigPath(), data, 0600)
}

// GetCFKey returns the effective CF key using priority: env > project config.
func GetCFKey() string {
	if k := ReadEnvCFKey(); k != "" {
		return k
	}
	if cfg, err := ReadProjectConfig(); err == nil && cfg != nil && cfg.CFKey != "" {
		return cfg.CFKey
	}
	return ""
}

// LoadDotEnv reads key=value pairs from the given file using godotenv and
// applies any that are not already present in the process environment. Existing
// values win so the operator can still override per-shell. A missing file is
// not an error.
// LoadDotEnv reads key=value pairs from the given file and applies them to
// the process environment when not already set. Values are NOT expanded for
// shell-style $VAR references, so a CF key like "$2a$10$..." is preserved
// verbatim (godotenv's default parser would otherwise eat the "$2" positional
// reference). Surrounding single or double quotes are stripped. Existing
// process env values win so the operator can still override per-shell. A
// missing file is not an error.
func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(line[:eq])
		v := strings.TrimSpace(line[eq+1:])
		if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		}
		if _, already := os.LookupEnv(k); !already {
			_ = os.Setenv(k, v)
		}
	}
	return nil
}
