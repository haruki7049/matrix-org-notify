package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadMessage(t *testing.T) {
	t.Run("from args", func(t *testing.T) {
		got, err := readMessage([]string{"hello", "world"}, strings.NewReader(""))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "hello world" {
			t.Fatalf("got %q, want %q", got, "hello world")
		}
	})

	t.Run("from stdin", func(t *testing.T) {
		got, err := readMessage(nil, strings.NewReader("piped message\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "piped message" {
			t.Fatalf("got %q, want %q", got, "piped message")
		}
	})
}

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

func TestSendMessage(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	var gotBody sendMessageRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")

		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"event_id":"$abc123"}`))
	}))
	defer server.Close()

	cfg := config{
		Homeserver:  server.URL,
		RoomID:      "!room:example.org",
		AccessToken: "test-token",
	}

	if err := sendMessage(server.Client(), cfg, "hello matrix"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if !strings.HasPrefix(gotPath, "/_matrix/client/v3/rooms/!room:example.org/send/m.room.message/") {
		t.Errorf("path = %q, unexpected prefix", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-token")
	}
	if gotBody.MsgType != "m.text" || gotBody.Body != "hello matrix" {
		t.Errorf("body = %+v, unexpected content", gotBody)
	}
}

func TestSendMessageErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errcode":"M_FORBIDDEN"}`))
	}))
	defer server.Close()

	cfg := config{Homeserver: server.URL, RoomID: "!room:example.org", AccessToken: "bad-token"}

	err := sendMessage(server.Client(), cfg, "hello")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error = %q, want it to mention status 403", err.Error())
	}
}

func TestRunNoMessage(t *testing.T) {
	err := run(nil, strings.NewReader(""), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected an error when no message is provided")
	}
}

func TestRunHelp(t *testing.T) {
	for _, name := range []string{"-h", "--help"} {
		var out bytes.Buffer
		err := run([]string{name}, strings.NewReader(""), &out)
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("err for %q = %v, want flag.ErrHelp", name, err)
		}
		if !strings.Contains(out.String(), "Usage:") {
			t.Errorf("output for %q = %q, want it to contain usage text", name, out.String())
		}
	}
}
