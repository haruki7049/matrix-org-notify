package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const defaultHomeserver = "https://matrix-client.matrix.org"

type config struct {
	Homeserver  string
	RoomID      string
	AccessToken string
}

const usageHeader = `matrix-org-notify sends a notification message to a Matrix room.

Usage:
  matrix-org-notify [flags] [message...]
  echo "message" | matrix-org-notify

The message is taken from the command-line arguments, or from stdin when
no arguments are given.

Configuration (Homeserver, Room ID, Access Token) is read from a JSON
config file (~/.config/matrix-org-notify/config.json on Linux/macOS,
%APPDATA%\matrix-org-notify\config.json on Windows) and can be overridden
with the MATRIX_HOMESERVER, MATRIX_ROOM_ID, and MATRIX_ACCESS_TOKEN
environment variables. See README.md for details.

Flags:
`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "matrix-org-notify:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("matrix-org-notify", flag.ContinueOnError)
	fs.SetOutput(stdout)
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), usageHeader)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	message, err := readMessage(fs.Args(), stdin)
	if err != nil {
		return err
	}
	if message == "" {
		return fmt.Errorf("no message provided (pass it as an argument or via stdin)")
	}

	cfg, err := loadConfig(configFilePath(), os.Getenv)
	if err != nil {
		return err
	}
	if cfg.RoomID == "" {
		return fmt.Errorf("room ID is not set (MATRIX_ROOM_ID or config file ROOM_ID)")
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("access token is not set (MATRIX_ACCESS_TOKEN or config file ACCESS_TOKEN)")
	}

	if err := sendMessage(http.DefaultClient, cfg, message); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "notification sent")
	return nil
}

// readMessage returns the message from CLI arguments, falling back to stdin
// when no arguments are given and stdin is not an interactive terminal.
func readMessage(args []string, stdin io.Reader) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}

	if f, ok := stdin.(*os.File); ok {
		info, err := f.Stat()
		if err == nil && (info.Mode()&os.ModeCharDevice) != 0 {
			return "", nil
		}
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("reading message from stdin: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
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

// configFile mirrors the JSON config file schema documented in README.md.
type configFile struct {
	Homeserver  string `json:"homeserver"`
	RoomID      string `json:"room_id"`
	AccessToken string `json:"access_token"`
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

type sendMessageRequest struct {
	MsgType string `json:"msgtype"`
	Body    string `json:"body"`
}

// sendMessage delivers message to cfg.RoomID via the Matrix Client-Server
// API: PUT /_matrix/client/v3/rooms/{roomId}/send/m.room.message/{txnId}.
func sendMessage(client *http.Client, cfg config, message string) error {
	txnID := fmt.Sprintf("%d", time.Now().UnixNano())

	url := fmt.Sprintf(
		"%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s",
		strings.TrimRight(cfg.Homeserver, "/"),
		urlPathEscape(cfg.RoomID),
		urlPathEscape(txnID),
	)

	payload, err := json.Marshal(sendMessageRequest{MsgType: "m.text", Body: message})
	if err != nil {
		return fmt.Errorf("encoding message payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request to matrix homeserver: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("matrix homeserver returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func urlPathEscape(s string) string {
	return strings.NewReplacer(
		"%", "%25",
		"/", "%2F",
		"#", "%23",
		"?", "%3F",
		" ", "%20",
	).Replace(s)
}
