//go:build integration

package queue_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danrichardson/pdarr/internal/db"
	"github.com/danrichardson/pdarr/internal/queue"
	"github.com/danrichardson/pdarr/internal/testutil"
)

func TestOriginalsGCSweep(t *testing.T) {
	database := testutil.NewTestDB(t)
	dir := t.TempDir()

	// Create a fake held original file.
	heldPath := filepath.Join(dir, ".processed", "media", "test.mkv")
	if err := os.MkdirAll(filepath.Dir(heldPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(heldPath, []byte("fake original"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Insert a job so we can create an originals record.
	jobID, err := database.InsertJob(&db.Job{
		SourcePath:     "/media/test.mkv",
		SourceSize:     1000,
		SourceCodec:    "h264",
		SourceDuration: 30,
		SourceBitrate:  8_000_000,
		Status:         db.JobStaged,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Insert an originals record already expired.
	_, err = database.InsertOriginal(&db.OriginalRecord{
		JobID:        jobID,
		OriginalPath: "/media/test.mkv",
		HeldPath:     heldPath,
		OutputPath:   "/media/test.h265.mkv",
		OriginalSize: 1000,
		OutputSize:   600,
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify file exists before GC.
	if _, err := os.Stat(heldPath); os.IsNotExist(err) {
		t.Fatal("held file should exist before GC")
	}

	gc := queue.NewOriginalsGC(database, testLog(t))
	gc.Sweep()

	// File should be deleted.
	if _, err := os.Stat(heldPath); !os.IsNotExist(err) {
		t.Error("held file should be deleted after GC sweep")
	}

	// No expired records should remain.
	records, err := database.ExpiredOriginals()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 expired records after GC, got %d", len(records))
	}
}
