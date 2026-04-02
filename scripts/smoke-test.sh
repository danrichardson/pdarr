#!/usr/bin/env bash
# SQZARR smoke test — starts daemon, adds directory, triggers scan,
# waits for one completed job, confirms DB state. Exits 0 on success.
set -euo pipefail

BINARY="${SQZARR_BINARY:-./sqzarr}"
TIMEOUT=120  # seconds to wait for a job to complete

if [[ ! -x "$BINARY" ]]; then
    echo "error: $BINARY not found. Run 'make build' first." >&2
    exit 1
fi
if ! command -v ffmpeg &>/dev/null; then
    echo "error: ffmpeg not on PATH" >&2
    exit 1
fi

# Create temp workspace
WORKDIR=$(mktemp -d)
trap "rm -rf $WORKDIR; kill $SQZARR_PID 2>/dev/null || true" EXIT

MEDIA_DIR="$WORKDIR/media"
DATA_DIR="$WORKDIR/data"
mkdir -p "$MEDIA_DIR" "$DATA_DIR"

# Create a test H.264 file aged > 7 days
TEST_FILE="$MEDIA_DIR/test_h264.mkv"
ffmpeg -y -f lavfi -i "testsrc=duration=30:size=1280x720:rate=25" \
    -c:v libx264 -b:v 8000k "$TEST_FILE" -loglevel error
touch -d "10 days ago" "$TEST_FILE"

# Write config
CONFIG="$WORKDIR/sqzarr.toml"
cat > "$CONFIG" <<EOF
[server]
host = "127.0.0.1"
port = 19876
data_dir = "$DATA_DIR"

[scanner]
interval_hours = 1
worker_concurrency = 1

[safety]
quarantine_enabled = true
quarantine_retention_days = 10
disk_free_pause_gb = 0
EOF

echo "Starting sqzarr..."
LIBVA_DRIVER_NAME=iHD "$BINARY" serve -config "$CONFIG" > "$WORKDIR/sqzarr.log" 2>&1 &
SQZARR_PID=$!
sleep 2

# Add media directory via API
curl -sf -X POST http://127.0.0.1:19876/api/v1/directories \
    -H 'Content-Type: application/json' \
    -d "{\"path\":\"$MEDIA_DIR\",\"min_age_days\":7,\"max_bitrate\":4000000,\"min_size_mb\":0}" \
    > /dev/null

# Trigger scan
curl -sf -X POST http://127.0.0.1:19876/api/v1/scan > /dev/null

# Wait for a completed job
echo "Waiting for job to complete (timeout: ${TIMEOUT}s)..."
ELAPSED=0
while true; do
    STATUS=$(curl -sf "http://127.0.0.1:19876/api/v1/jobs?status=done&limit=1" 2>/dev/null || echo "[]")
    COUNT=$(echo "$STATUS" | python3 -c "import json,sys; d=json.load(sys.stdin); print(len(d) if d else 0)" 2>/dev/null || echo 0)
    if [[ "$COUNT" -gt 0 ]]; then
        echo "Job completed successfully!"
        break
    fi
    if [[ $ELAPSED -ge $TIMEOUT ]]; then
        echo ""
        echo "TIMEOUT: no completed job after ${TIMEOUT}s" >&2
        echo "sqzarr log:" >&2
        cat "$WORKDIR/sqzarr.log" >&2
        exit 1
    fi
    sleep 5
    ELAPSED=$((ELAPSED + 5))
    echo -n "."
done

# Verify bytes_saved is non-null in the done job
BYTES_SAVED=$(curl -sf "http://127.0.0.1:19876/api/v1/jobs?status=done&limit=1" | \
    python3 -c "import json,sys; d=json.load(sys.stdin); print(d[0]['BytesSaved']['Int64'] if d and d[0]['BytesSaved']['Valid'] else 0)")
if [[ "$BYTES_SAVED" -le 0 ]]; then
    echo "FAIL: bytes_saved is 0 or null" >&2
    exit 1
fi

echo "Smoke test passed. Bytes saved: $BYTES_SAVED"
exit 0
