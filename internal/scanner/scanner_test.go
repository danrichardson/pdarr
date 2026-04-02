//go:build integration

package scanner_test

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/danrichardson/pdarr/internal/db"
	"github.com/danrichardson/pdarr/internal/scanner"
	"github.com/danrichardson/pdarr/internal/testutil"
)

func testLog(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.Default()
}

func TestScanFindsH264File(t *testing.T) {
	database := testutil.NewTestDB(t)
	dir := t.TempDir()

	clipPath := testutil.MakeTestClip(t, dir, 30)
	past := time.Now().Add(-10 * 24 * time.Hour)
	os.Chtimes(clipPath, past, past)

	dirID, err := database.InsertDirectory(&db.Directory{
		Path:       dir,
		Enabled:    true,
		MinAgeDays: 7,
		MaxBitrate: 4_000_000,
		MinSizeMB:  0,
	})
	if err != nil {
		t.Fatal(err)
	}

	dbDir, _ := database.GetDirectory(dirID)
	s := scanner.New(database, testLog(t))
	result, err := s.ScanDirectory(context.Background(), dbDir)
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesQueued != 1 {
		t.Errorf("expected 1 queued, got %d", result.FilesQueued)
	}

	jobs, err := database.ListJobs(db.JobPending, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 pending job, got %d", len(jobs))
	}
	if jobs[0].SourcePath != clipPath {
		t.Errorf("expected source path %s, got %s", clipPath, jobs[0].SourcePath)
	}
}

func TestScanSkipsHEVC(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}

	database := testutil.NewTestDB(t)
	dir := t.TempDir()

	clipPath := filepath.Join(dir, "hevc.mkv")
	cmd := exec.Command("ffmpeg", "-y",
		"-f", "lavfi",
		"-i", "testsrc=duration=5:size=640x480:rate=25",
		"-c:v", "libx265", "-crf", "28",
		clipPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("cannot create HEVC test clip: %v\n%s", err, out)
	}

	past := time.Now().Add(-10 * 24 * time.Hour)
	os.Chtimes(clipPath, past, past)

	dirID, _ := database.InsertDirectory(&db.Directory{
		Path:       dir,
		Enabled:    true,
		MinAgeDays: 7,
		MaxBitrate: 4_000_000,
	})
	dbDir, _ := database.GetDirectory(dirID)

	s := scanner.New(database, testLog(t))
	result, _ := s.ScanDirectory(context.Background(), dbDir)

	if result.FilesQueued != 0 {
		t.Errorf("expected 0 queued for HEVC file, got %d", result.FilesQueued)
	}
}

func TestScanSkipsNewFile(t *testing.T) {
	database := testutil.NewTestDB(t)
	dir := t.TempDir()

	testutil.MakeTestClip(t, dir, 10)
	// mtime is now — must fail the min_age_days=7 check.

	dirID, _ := database.InsertDirectory(&db.Directory{
		Path:       dir,
		Enabled:    true,
		MinAgeDays: 7,
		MaxBitrate: 4_000_000,
	})
	dbDir, _ := database.GetDirectory(dirID)

	s := scanner.New(database, testLog(t))
	result, _ := s.ScanDirectory(context.Background(), dbDir)

	if result.FilesQueued != 0 {
		t.Errorf("expected 0 queued for new file, got %d", result.FilesQueued)
	}
}

func TestScanDeduplicates(t *testing.T) {
	database := testutil.NewTestDB(t)
	dir := t.TempDir()

	clipPath := testutil.MakeTestClip(t, dir, 30)
	past := time.Now().Add(-10 * 24 * time.Hour)
	os.Chtimes(clipPath, past, past)

	dirID, _ := database.InsertDirectory(&db.Directory{
		Path:       dir,
		Enabled:    true,
		MinAgeDays: 7,
		MaxBitrate: 4_000_000,
	})
	dbDir, _ := database.GetDirectory(dirID)
	s := scanner.New(database, testLog(t))

	s.ScanDirectory(context.Background(), dbDir)
	result, _ := s.ScanDirectory(context.Background(), dbDir)

	if result.FilesQueued != 0 {
		t.Errorf("expected 0 queued on second scan, got %d", result.FilesQueued)
	}
}
