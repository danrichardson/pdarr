# PDARR

Self-hosted GPU-accelerated media transcoder. Scans your library, finds files that are too big, and quietly compresses them using your GPU. No Docker, no subscriptions, no bullshit — a single Go binary and a clean web UI.

**Target hardware:** Intel Quick Sync (VAAPI), Apple VideoToolbox, NVIDIA NVENC, or CPU fallback.

---

## What it does

1. Watches configured media directories on a schedule (default: every 6 hours)
2. Finds video files that exceed your bitrate/size threshold and aren't already HEVC
3. Transcodes them to HEVC using GPU hardware encoding
4. Verifies the output (smaller file, duration match) before replacing the original
5. Moves the original to a quarantine folder for 10 days before permanent deletion
6. Optionally triggers a Plex library refresh after each replacement

---

## Requirements

- **Linux:** `ffmpeg` 4.x+ with VAAPI support
- **macOS:** `ffmpeg` 4.x+ (Homebrew: `brew install ffmpeg`)
- `ffprobe` on PATH (included with ffmpeg)

---

## Proxmox LXC Setup (Intel VAAPI)

### 1. Verify VAAPI on host

```bash
vainfo
# Must show: VAProfileHEVCMain : VAEntrypointEncSlice
```

### 2. LXC container config

Required additions to `/etc/pve/lxc/<ctid>.conf`:

```
# GPU passthrough
lxc.mount.entry: /dev/dri/ dev/dri/ none bind,optional,create=dir
lxc.cgroup2.devices.allow: c 226:* rwm

# UID/GID maps — punch hole for UID 1000 and render/video groups
lxc.idmap: u 0 100000 1000
lxc.idmap: u 1000 1000 1
lxc.idmap: u 1001 101001 64535
lxc.idmap: g 0 100000 44
lxc.idmap: g 44 44 1
lxc.idmap: g 45 100045 55
lxc.idmap: g 100 100 1
lxc.idmap: g 101 100101 3
lxc.idmap: g 104 104 1
lxc.idmap: g 105 100105 65431
```

Host `/etc/subuid` must include:
```
root:100000:65536
root:1000:1
```

Host `/etc/subgid` must include:
```
root:100000:65536
root:44:1
root:100:1
root:104:1
```

### 3. Inside the container (Debian 13)

```bash
# Enable non-free for HEVC encode support
# Edit /etc/apt/sources.list.d/debian.sources — add non-free non-free-firmware to Components
apt update
apt install ffmpeg vainfo intel-media-va-driver-non-free

# Verify HEVC encode works
LIBVA_DRIVER_NAME=iHD vainfo | grep "HEVC.*EncSlice"
```

### 4. Install PDARR

```bash
# Download the latest release
curl -L https://github.com/danrichardson/pdarr/releases/latest/download/pdarr-linux-amd64 \
  -o /usr/local/bin/pdarr && chmod +x /usr/local/bin/pdarr

# Create config
mkdir -p /etc/pdarr /var/lib/pdarr
cp pdarr.toml.example /etc/pdarr/pdarr.toml
# Edit /etc/pdarr/pdarr.toml — set data_dir, add directories

# Install and start systemd service
cp scripts/pdarr.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now pdarr
```

### 5. Environment variable

The service unit sets `LIBVA_DRIVER_NAME=iHD`. If you're running manually:

```bash
LIBVA_DRIVER_NAME=iHD pdarr serve -config /etc/pdarr/pdarr.toml
```

---

## macOS Setup (Apple Silicon)

Requires macOS 13+ on M-series hardware.

```bash
brew install ffmpeg

# Build from source or download darwin-arm64 release binary
make build-darwin   # produces dist/pdarr-darwin-arm64

# Install
./scripts/install-macos.sh

# Edit config
nano /Users/Shared/pdarr/pdarr.toml

# Start service
sudo launchctl kickstart -k system/com.pdarr.agent
```

---

## Configuration

Copy `pdarr.toml.example` to `/etc/pdarr/pdarr.toml` and edit:

```toml
[server]
host = "127.0.0.1"   # change to 0.0.0.0 for LAN access
port = 8080
data_dir = "/var/lib/pdarr"

[scanner]
interval_hours = 6
worker_concurrency = 1

[safety]
quarantine_enabled = true
quarantine_retention_days = 10
disk_free_pause_gb = 50

[plex]
enabled = false
base_url = "http://192.168.1.10:32400"
token = "your-plex-token"

[auth]
# Leave empty to disable authentication
# To set: run `pdarr hash-password` and paste output here
password_hash = ""
jwt_secret = ""
```

### Adding directories via CLI (before UI is set up)

```bash
# Add a directory directly via the API after starting the service:
curl -s -X POST http://localhost:8080/api/v1/directories \
  -H 'Content-Type: application/json' \
  -d '{"path":"/media/Videos","min_age_days":7,"max_bitrate":4000000,"min_size_mb":500}'
```

Or use the web UI at `http://localhost:8080`.

---

## CLI Reference

```
pdarr serve              Start the HTTP server and worker daemon
pdarr scan-once          Run a single scan pass and exit
pdarr scan-once --dry-run  Scan without enqueuing (preview only)
pdarr restore <job-id>   Restore original from quarantine
pdarr hash-password      Generate a bcrypt hash for pdarr.toml
```

---

## Building from source

```bash
# Backend only
make build

# Frontend + backend (embedded)
make all

# Release binaries (linux/amd64 + darwin/arm64)
make release
```

**Requirements:** Go 1.22+, Node 20+, ffmpeg (for integration tests)

---

## Admin panel

Navigate to `http://<host>:8080` after starting the service.

- **Dashboard** — encoder status, space saved, active job progress, disk space bar
- **Queue** — running job with live progress, pending list with cancel
- **History** — paginated job history with expandable error rows and retry
- **Directories** — add/edit/delete watched directories with per-directory rules
- **Settings** — encoder info, config file reference, auth setup

---

## Security notes

- The admin panel is bound to `127.0.0.1` by default — not exposed to LAN without changing `host`
- Authentication is optional; set `password_hash` to enable it
- All file paths in API requests are validated against configured directory roots (path traversal prevention)
- Config file should not be world-readable: `chmod 600 /etc/pdarr/pdarr.toml`

---

## FAQ

**Q: Will it delete my files?**  
Originals are moved to quarantine for `quarantine_retention_days` (default 10) before deletion. Use `pdarr restore <job-id>` to recover any file within that window.

**Q: What if the output is larger than the input?**  
The verifier rejects it — the original is restored and the job is marked failed. Nothing is lost.

**Q: Can I run multiple workers?**  
Set `worker_concurrency` up to 8. Most GPU hardware handles one HEVC encode stream at a time; running more may not help and will increase temp disk usage.

**Q: The Plex token?**  
Open Plex Web, sign in, navigate to a library item, open DevTools → Network, filter by your server IP. Any request URL will contain `X-Plex-Token=<your-token>`.
