# matrix-org-notify

`matrix-org-notify` is a lightweight, cross-platform CLI tool written in Go for sending notification messages to Matrix rooms (e.g., `matrix.org`). It is designed for fast alerting and seamless integration with developer workflows such as Claude Code hooks, CI/CD pipelines, long-running scripts, and cron jobs.

## Features

- **Zero External Dependencies**: Built strictly using Go's standard library for fast builds and minimal footprint.
- **Cross-Platform**: Runs natively on Linux, macOS, and Windows.
- **Security-First**: Secrets (Access Tokens and Room IDs) remain safely on your local machine and are never tracked in version control.
- **Nix Flake Support**: Can be executed directly without manual installation via `nix run`.
- **Flexible Input**: Accepts notification messages via command-line arguments or piped through standard input (`stdin`).

## Setup & Configuration

To use `matrix-org-notify`, you need:

1. A Matrix homeserver URL (default: `https://matrix-client.matrix.org`).
1. A target Matrix Room ID (e.g. `!abcdefgh:matrix.org`).
1. An Access Token from an account joined to that room (e.g. a dedicated bot account).

> [!IMPORTANT]
> The target Matrix room should have **End-to-End Encryption (E2EE) disabled**, as this lightweight tool talks directly to the standard Matrix Client-Server HTTP API without heavy encryption client dependencies.

### Configuration File

Create a configuration file in your user config directory:

- **Linux / macOS**: `~/.config/matrix-notify/config`
- **Windows**: `%APPDATA%\matrix-notify\config`

Example content:

```bash
HOMESERVER="https://matrix-client.matrix.org"
ROOM_ID="!your_room_id:matrix.org"
ACCESS_TOKEN="syt_your_access_token_here"
```

Restrict permissions on the file to prevent unauthorized access:

```bash
chmod 600 ~/.config/matrix-notify/config
```

### Environment Variables (Optional)

You can also configure or override settings via environment variables:

- `MATRIX_HOMESERVER`
- `MATRIX_ROOM_ID`
- `MATRIX_ACCESS_TOKEN`

## Usage

### Direct Command

```bash
# Pass message as an argument
matrix-org-notify "Build finished successfully!"

# Or pass message through stdin
echo "Task completed!" | matrix-org-notify
```

### Running with Nix

Run directly without cloning or building manually:

```bash
nix run github:haruki7049/matrix-org-notify -- "Hello from Nix!"
```

## Claude Code Integration

You can configure Claude Code to notify you on Matrix when a task completes or when it waits for your input or tool permission by adding hooks to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "Notification": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "matrix-org-notify '⚠️ [Claude Code] Waiting for user approval/input'"
          }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "matrix-org-notify '✅ [Claude Code] Task finished'"
          }
        ]
      }
    ]
  }
}
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
