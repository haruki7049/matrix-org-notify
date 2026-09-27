package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunNoMessage(t *testing.T) {
	err := run(nil, strings.NewReader(""), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected an error when no message is provided")
	}
}

func TestRunUsesConfigFlag(t *testing.T) {
	for _, flagName := range []string{"-c", "--config", "-config"} {
		t.Run(flagName, func(t *testing.T) {
			var gotAuth string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				_, _ = w.Write([]byte(`{"event_id":"$abc123"}`))
			}))
			defer server.Close()

			dir := t.TempDir()
			path := filepath.Join(dir, "custom-config.json")
			content := fmt.Sprintf(
				`{"homeserver": %q, "room_id": "!room:example.org", "access_token": "flag-token"}`,
				server.URL,
			)
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatalf("writing config file: %v", err)
			}

			var out bytes.Buffer
			err := run([]string{flagName, path, "hello"}, strings.NewReader(""), &out)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotAuth != "Bearer flag-token" {
				t.Errorf("Authorization = %q, want the token from the flagged config file", gotAuth)
			}
		})
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
