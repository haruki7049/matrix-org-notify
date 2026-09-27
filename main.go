// Command matrix-org-notify sends a notification message to a Matrix room.
//
// Run matrix-org-notify -h for usage, configuration file location, and
// environment variable overrides.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

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

Example config.json:
  {
    "homeserver": "https://matrix-client.matrix.org",
    "room_id": "!your_room_id",
    "access_token": "syt_your_access_token_here"
  }

room_id is the exact Matrix room ID (starting with "!") as returned by
your homeserver, e.g. via a client's room settings. Depending on the
homeserver and room version, it may or may not include a ":server_name"
suffix; use it exactly as given, do not append one yourself.

Flags:
`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		if errors.Is(err, flag.ErrHelp) {
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
		return errors.New("no message provided (pass it as an argument or via stdin)")
	}

	cfg, err := loadConfig(configFilePath(), os.Getenv)
	if err != nil {
		return err
	}
	if cfg.RoomID == "" {
		return errors.New("room ID is not set (MATRIX_ROOM_ID or config file room_id)")
	}
	if cfg.AccessToken == "" {
		return errors.New("access token is not set (MATRIX_ACCESS_TOKEN or config file access_token)")
	}

	if err := sendMessage(httpClient, cfg, message); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "notification sent")
	return nil
}
