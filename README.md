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
1. A target Matrix Room ID, exactly as given by your homeserver (e.g. `!k0_TCUEmo5lIZChur5RovqVqrc2hlZ7GDV9Q5uSGI_Y`). Depending on the homeserver and room version, it may or may not include a `:server_name` suffix (e.g. `!abcdefgh:matrix.org`) — do not append one yourself.
1. An Access Token from an account joined to that room (e.g. a dedicated bot account).

> [!IMPORTANT]
> The target Matrix room should have **End-to-End Encryption (E2EE) disabled**, as this lightweight tool talks directly to the standard Matrix Client-Server HTTP API without heavy encryption client dependencies.

### Configuration File

Create a JSON configuration file in your user config directory:

- **Linux / macOS**: `~/.config/matrix-org-notify/config.json`
- **Windows**: `%APPDATA%\matrix-org-notify\config.json`

Example content:

```json
{
  "homeserver": "https://matrix-client.matrix.org",
  "room_id": "!your_room_id",
  "access_token": "syt_your_access_token_here"
}
```

Restrict permissions on the file to prevent unauthorized access:

```bash
chmod 600 ~/.config/matrix-org-notify/config.json
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

### Protecting the Access Token from Claude Code Itself

If you use `matrix-org-notify` from a Claude Code hook (see below), prefer the **config file** over the `MATRIX_ACCESS_TOKEN` environment variable for that use case. An environment variable set in the shell Claude Code runs in can be read back by the agent itself (e.g. by running `env`), since Claude Code's `Bash` tool shares that same environment. A value that only lives in `~/.config/matrix-org-notify/config.json` is not exposed this way — only the `matrix-org-notify` binary reads it.

This is not a hard guarantee: an agent with unrestricted `Bash` access can still read the file directly (`cat`, `head`, a script that opens it, etc.). As defense in depth, you can add deny rules to `~/.claude/settings.json` to keep Claude Code from touching the file or dumping the environment:

```json
{
  "permissions": {
    "deny": [
      "Read(~/.config/matrix-org-notify/**)",
      "Bash(env)",
      "Bash(printenv)",
      "Bash(cat ~/.config/matrix-org-notify/config.json)"
    ]
  }
}
```

Treat this as a speed bump, not a security boundary: `Bash` deny rules match specific command text and can be bypassed by an absolute path, a different shell wrapper, or another program that reads the file (`less`, `python -c "..."`, etc.). The `chmod 600` step above (OS-level file permissions) is the actual enforcement; the deny rules just make an accidental or casual read less likely.

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
