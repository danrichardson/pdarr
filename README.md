# PDARR

Self-hosted GPU-accelerated media transcoder. Scans your library, finds files that are too big, and quietly compresses them using your GPU. No Docker, no subscriptions, no bullshit. Single Go binary and a clean web UI.

**Target hardware:** Intel Quick Sync (VAAPI), Apple VideoToolbox, NVIDIA NVENC, or CPU fallback.

---

## What it does

1. Watches configured media directories on a schedule (default: every 6 hours)
2. Finds video files that exceed your bitrate/size threshold (default: ~1 GB/hr)
3. Transcodes them to HEVC using GPU hardware encoding. Any codec qualifies, including HEVC files that are still too large
4. Verifies the output (smaller file, duration match) before replacing the original
5. Optionally holds the original for a configurable retention period before permanent deletion
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

The container can and should remain **unprivileged** (`unprivileged: 1`). Do not change this.

Required additions for GPU passthrough:

```
# GPU passthrough
lxc.mount.entry: /dev/dri/ dev/dri/ none bind,optional,create=dir
lxc.cgroup2.devices.allow: c 226:* rwm
```

### 3. Grant write access to the media directory (host-side, one command)

In an unprivileged LXC, UID 0 inside the container maps to UID 100000 on the Proxmox host. Your media directories need to be owned by that host UID so pdarr can write temp files next to source files.

**Run this on the PVE host** (not inside the container), targeting only the Videos directory (not the whole pool):

```bash
chown -R 100000:100000 /storePool/subvol-250-disk-0/srv/storage/Videos
```

This is safe and scoped: pdarr can only write to the directories it's already watching. The rest of your NAS is unaffected. The container stays unprivileged.

### 4. Service user

The provided `pdarr.service` runs as root inside the container. Combined with the `chown` above, this is sufficient for write access.

### 4. Inside the container (Debian 13)

```bash
# Enable non-free for HEVC encode support
# Edit /etc/apt/sources.list.d/debian.sources - add non-free non-free-firmware to Components
apt update
apt install ffmpeg vainfo intel-media-va-driver-non-free

# Verify HEVC encode works
LIBVA_DRIVER_NAME=iHD vainfo | grep "HEVC.*EncSlice"
```

### 5. Install PDARR

```bash
# Download the latest release
curl -L https://github.com/danrichardson/pdarr/releases/latest/download/pdarr-linux-amd64 \
  -o /usr/local/bin/pdarr && chmod +x /usr/local/bin/pdarr

# Create config
mkdir -p /etc/pdarr /var/lib/pdarr
cp pdarr.toml.example /etc/pdarr/pdarr.toml
# Edit /etc/pdarr/pdarr.toml - set data_dir, add directories

# Install and start systemd service
cp scripts/pdarr.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now pdarr
```

### 6. Go PATH (if building from source on the server)

If Go is installed system-wide (e.g. `/usr/local/go`), make sure `/etc/profile.d/go.sh` contains only:

```bash
export PATH=$PATH:/usr/local/go/bin
```

Do not let any Windows or Git Bash PATH values end up in this file. It will break login shells.

### 7. Environment variable

The service unit sets `LIBVA_DRIVER_NAME=iHD`. If running manually:

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

**Note:** There is no `disk_free_pause_gb` setting. PDARR never pauses the whole queue based on disk space. It checks per-job whether there is enough temp space (1.2× the source file size) before starting each transcode. If space is insufficient for a specific file it skips that job and moves on.

### Default bitrate threshold

Files are transcoded when their average bitrate exceeds `max_bitrate` (per-directory setting). The default is **2,222,000 bits/sec (~1 GB/hour)**. A 5 GB file for a 1-hour show is ~11 Mbps, well above threshold. A well-encoded 1080p HEVC file at 4 Mbps would be skipped.

The codec does not matter. Oversized HEVC, AV1, and H.264 files are all eligible.

### Adding directories via CLI (before UI is set up)

```bash
curl -s -X POST http://localhost:8080/api/v1/directories \
  -H 'Content-Type: application/json' \
  -d '{"path":"/media/Videos","min_age_days":7,"max_bitrate":2222000,"min_size_mb":500}'
```

Or use the web UI at `http://localhost:8080`.

---

## Database migrations

If upgrading an existing installation, run any required schema changes before starting the new binary. Known migrations:

| Version | Change | Command |
|---------|--------|---------|
| Post-phase-4 → current | Added `file_size` column to `quarantine` table | `sqlite3 /var/lib/pdarr/pdarr.db 'ALTER TABLE quarantine ADD COLUMN file_size INTEGER NOT NULL DEFAULT 0;'` |

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

If you don't have Go on your build machine, you can build directly on the server:

```bash
# On the server, after copying source:
cd /opt/pdarr-src
go mod tidy
go build -trimpath -ldflags='-s -w' -o /usr/local/bin/pdarr ./cmd/pdarr/
```

---

## Admin panel

Navigate to `http://<host>:8080` after starting the service.

- **Dashboard**: encoder in use, space saved, jobs done/failed, active job progress, disk space (measured at your first configured media directory)
- **Queue**: manual file enqueue with filesystem browser, running job with live progress, pending list with cancel
- **History**: paginated job history with expandable error detail (full ffmpeg output on failure), retry button, clear history
- **Directories**: add/edit/delete watched directories with per-directory rules; filesystem browser for path selection
- **Processed**: originals held after successful transcode; restore (roll back) or purge (delete early) per file
- **Settings**: encoder info, scan interval, worker concurrency, retention period, Plex config - all editable live

---

## Security notes

- The admin panel is bound to `127.0.0.1` by default. Not exposed to LAN without changing `host`
- Authentication is optional; set `password_hash` to enable it
- All file paths in API requests are validated against configured directory roots (path traversal prevention)
- Config file should not be world-readable: `chmod 600 /etc/pdarr/pdarr.toml`

---

## FAQ

**Q: Will it delete my files?**  
When `quarantine_enabled = true` (the default), originals are held for `quarantine_retention_days` (default 10) before deletion. You can restore any file within that window from the Processed page or via `pdarr restore <job-id>`.

**Q: What if the output is larger than the input?**  
The verifier rejects it. The original is restored and the job is marked failed. Nothing is lost.

**Q: My jobs all fail with "Permission denied".**  
pdarr writes a temp file next to the source file before atomically replacing it. On Proxmox with a ZFS-backed media bindmount, the LXC's UID mapping means even root inside the container maps to a non-root host UID, which ZFS rejects. The fix is a single `chown` on the PVE host targeting only your Videos directory (see step 3 of the Proxmox LXC Setup section). The container stays unprivileged.

**Q: Can I run multiple workers?**  
Set `worker_concurrency` up to 8. Most GPU hardware handles one HEVC encode stream at a time; running more may not help and will increase temp disk usage.

**Q: The Plex token?**  
Open Plex Web, sign in, navigate to a library item, open DevTools → Network, filter by your server IP. Any request URL will contain `X-Plex-Token=<your-token>`.
