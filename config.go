package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const defaultHomeserver = "https://matrix-client.matrix.org"

// config holds the resolved settings needed to send a Matrix notification.
type config struct {
	Homeserver  string
	RoomID      string
	AccessToken string
}

// configFile mirrors the JSON config file schema documented in README.md.
type configFile struct {
	Homeserver  string `json:"homeserver"`
	RoomID      string `json:"room_id"`
	AccessToken string `json:"access_token"`
}

// configFilePath returns the path to the config file, following the
// convention documented in README.md: ~/.config/matrix-org-notify/config.json
// on Linux/macOS, and %APPDATA%\matrix-org-notify\config.json on Windows.
func configFilePath() string {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return ""
		}
		return filepath.Join(appData, "matrix-org-notify", "config.json")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "matrix-org-notify", "config.json")
}

// loadConfig reads the JSON config file (if present) and applies
// environment variable overrides on top of it.
func loadConfig(path string, getenv func(string) string) (config, error) {
	cfg := config{Homeserver: defaultHomeserver}

	if path != "" {
		file, err := parseConfigFile(path)
		if err != nil {
			return config{}, err
		}
		if file.Homeserver != "" {
			cfg.Homeserver = file.Homeserver
		}
		cfg.RoomID = file.RoomID
		cfg.AccessToken = file.AccessToken
	}

	if v := getenv("MATRIX_HOMESERVER"); v != "" {
		cfg.Homeserver = v
	}
	if v := getenv("MATRIX_ROOM_ID"); v != "" {
		cfg.RoomID = v
	}
	if v := getenv("MATRIX_ACCESS_TOKEN"); v != "" {
		cfg.AccessToken = v
	}

	return cfg, nil
}

// parseConfigFile parses the JSON config file. A missing file is not an
// error, as configuration may be supplied entirely via environment
// variables.
func parseConfigFile(path string) (configFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return configFile{}, nil
		}
		return configFile{}, fmt.Errorf("opening config file: %w", err)
	}

	var file configFile
	if err := json.Unmarshal(data, &file); err != nil {
		return configFile{}, fmt.Errorf("parsing config file: %w", err)
	}

	return file, nil
}
