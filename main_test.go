package main

import (
	"bytes"
	"encoding/json"
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
	path := filepath.Join(dir, "config")
	content := `
# comment line
HOMESERVER="https://example.org"
ROOM_ID='!room:example.org'
ACCESS_TOKEN=plain-token
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	values, err := parseConfigFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]string{
		"HOMESERVER":   "https://example.org",
		"ROOM_ID":      "!room:example.org",
		"ACCESS_TOKEN": "plain-token",
	}
	for k, v := range want {
		if values[k] != v {
			t.Errorf("values[%q] = %q, want %q", k, values[k], v)
		}
	}
}

func TestParseConfigFileMissing(t *testing.T) {
	values, err := parseConfigFile(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("got %v, want empty map", values)
	}
}

func TestLoadConfigEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	content := `
HOMESERVER="https://file.example.org"
ROOM_ID="!file:example.org"
ACCESS_TOKEN="file-token"
`
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
