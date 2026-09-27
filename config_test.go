package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
  "homeserver": "https://example.org",
  "room_id": "!room:example.org",
  "access_token": "plain-token"
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	got, err := parseConfigFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := configFile{
		Homeserver:  "https://example.org",
		RoomID:      "!room:example.org",
		AccessToken: "plain-token",
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseConfigFileMissing(t *testing.T) {
	got, err := parseConfigFile(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != (configFile{}) {
		t.Fatalf("got %+v, want zero value", got)
	}
}

func TestLoadConfigEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
  "homeserver": "https://file.example.org",
  "room_id": "!file:example.org",
  "access_token": "file-token"
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	env := map[string]string{
		"MATRIX_ROOM_ID": "!env:example.org",
	}
	getenv := func(key string) string { return env[key] }

	cfg, err := loadConfig(path, getenv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Homeserver != "https://file.example.org" {
		t.Errorf("Homeserver = %q, want file value to be kept", cfg.Homeserver)
	}
	if cfg.RoomID != "!env:example.org" {
		t.Errorf("RoomID = %q, want env override", cfg.RoomID)
	}
	if cfg.AccessToken != "file-token" {
		t.Errorf("AccessToken = %q, want file value to be kept", cfg.AccessToken)
	}
}

func TestLoadConfigDefaultsHomeserver(t *testing.T) {
	cfg, err := loadConfig(filepath.Join(t.TempDir(), "does-not-exist"), func(string) string { return "" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Homeserver != defaultHomeserver {
		t.Errorf("Homeserver = %q, want default %q", cfg.Homeserver, defaultHomeserver)
	}
}
