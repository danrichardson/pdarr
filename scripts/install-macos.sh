#!/usr/bin/env bash
# PDARR macOS installer — tested on macOS 14+ (Apple Silicon)
set -euo pipefail

BINARY="./pdarr-darwin-arm64"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/Users/Shared/pdarr"
LOG_DIR="/var/log/pdarr"
PLIST_SRC="$(dirname "$0")/com.pdarr.agent.plist"
PLIST_DEST="/Library/LaunchDaemons/com.pdarr.agent.plist"

if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "error: this script is for macOS only" >&2
    exit 1
fi

if [[ ! -f "$BINARY" ]]; then
    echo "error: $BINARY not found — build it first with 'make build-darwin'" >&2
    exit 1
fi

echo "Installing PDARR..."

# Binary
sudo install -m 755 "$BINARY" "$INSTALL_DIR/pdarr"
echo "  [ok] binary -> $INSTALL_DIR/pdarr"

# Config directory
sudo mkdir -p "$CONFIG_DIR"
if [[ ! -f "$CONFIG_DIR/pdarr.toml" ]]; then
    sudo tee "$CONFIG_DIR/pdarr.toml" > /dev/null <<TOML
[server]
host = "127.0.0.1"
port = 8080
data_dir = "/Users/Shared/pdarr"

[scanner]
interval_hours = 6
worker_concurrency = 1

[safety]
quarantine_enabled = true
quarantine_retention_days = 10
disk_free_pause_gb = 50

[plex]
enabled = false
base_url = ""
token = ""

[auth]
password_hash = ""
jwt_secret = ""
TOML
    echo "  [ok] config -> $CONFIG_DIR/pdarr.toml (edit before starting service)"
else
    echo "  [skip] config already exists at $CONFIG_DIR/pdarr.toml"
fi

# Log directory
sudo mkdir -p "$LOG_DIR"
echo "  [ok] log dir -> $LOG_DIR"

# LaunchDaemon plist
sudo install -m 644 "$PLIST_SRC" "$PLIST_DEST"
sudo launchctl load -w "$PLIST_DEST" 2>/dev/null || true
echo "  [ok] launchd -> $PLIST_DEST"

echo ""
echo "PDARR installed."
echo ""
echo "Edit $CONFIG_DIR/pdarr.toml to add your media directories."
echo "Then: sudo launchctl kickstart -k system/com.pdarr.agent"
echo ""
echo "Admin panel: http://localhost:8080"
