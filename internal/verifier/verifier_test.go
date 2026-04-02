//go:build integration

package verifier_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/danrichardson/sqzarr/internal/verifier"
)

func TestVerifyPassesValidOutput(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}

	dir := t.TempDir()
	input := filepath.Join(dir, "input.mkv")
	output := filepath.Join(dir, "output.mkv")

	// Large-ish H.264 input.
	run(t, "ffmpeg", "-y", "-f", "lavfi",
		"-i", "testsrc=duration=30:size=1280x720:rate=25",
		"-c:v", "libx264", "-b:v", "8000k", input)

	// Smaller output (lower bitrate).
	run(t, "ffmpeg", "-y", "-i", input,
		"-c:v", "libx264", "-b:v", "1000k", output)

	result, err := verifier.Verify(input, output, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Errorf("expected OK, got: %s", result.Reason)
	}
}

func TestVerifyFailsWhenOutputLarger(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}

	dir := t.TempDir()
	input := filepath.Join(dir, "input.mkv")
	output := filepath.Join(dir, "output.mkv")

	// Low-bitrate input.
	run(t, "ffmpeg", "-y", "-f", "lavfi",
		"-i", "testsrc=duration=10:size=320x240:rate=25",
		"-c:v", "libx264", "-b:v", "100k", input)

	// Higher-bitrate output — should fail verification.
	run(t, "ffmpeg", "-y", "-i", input,
		"-c:v", "libx264", "-b:v", "5000k", output)

	result, err := verifier.Verify(input, output, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Error("expected verification to fail (output larger than input)")
	}
}

func run(t *testing.T, name string, args ...string) {
	t.Helper()
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
