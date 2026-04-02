package transcoder

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ProgressFunc is called periodically with progress in [0.0, 1.0].
type ProgressFunc func(progress float64)

// Transcoder runs ffmpeg jobs using a detected encoder.
type Transcoder struct {
	encoder *Encoder
	log     *slog.Logger
	tempDir string
}

// New creates a Transcoder. If tempDir is empty, output files are written
// adjacent to the source file with a .pdarr-tmp suffix.
func New(enc *Encoder, tempDir string, log *slog.Logger) *Transcoder {
	return &Transcoder{encoder: enc, tempDir: tempDir, log: log}
}

// Encoder returns the active encoder.
func (t *Transcoder) Encoder() *Encoder {
	return t.encoder
}

// Run transcodes inputPath to a temp file, returning the temp output path.
// The caller is responsible for verifying and renaming the output.
func (t *Transcoder) Run(ctx context.Context, inputPath string, duration float64, onProgress ProgressFunc) (outputPath string, err error) {
	outputPath = t.tempOutputPath(inputPath)

	// Ensure temp dir exists.
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	args := append(t.encoder.BuildArgs(inputPath, outputPath),
		"-progress", "pipe:2",
	)
	// Insert progress pipe before the output path (last arg).
	// Rebuild: add -progress pipe:2 before final output path arg.
	args = rebuildWithProgress(t.encoder.BuildArgs(inputPath, outputPath))

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	// Set LIBVA_DRIVER_NAME for VAAPI.
	if t.encoder.Type == EncoderVAAPI {
		cmd.Env = append(os.Environ(), "LIBVA_DRIVER_NAME=iHD")
	} else {
		cmd.Env = os.Environ()
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start ffmpeg: %w", err)
	}

	// Parse progress from stderr.
	go func() {
		scanner := bufio.NewScanner(stderr)
		var currentTime float64
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "out_time_ms=") {
				ms, _ := strconv.ParseFloat(strings.TrimPrefix(line, "out_time_ms="), 64)
				currentTime = ms / 1_000_000
				if duration > 0 && onProgress != nil {
					pct := currentTime / duration
					if pct > 1 {
						pct = 1
					}
					onProgress(pct)
				}
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		os.Remove(outputPath)
		return "", fmt.Errorf("ffmpeg: %w", err)
	}

	return outputPath, nil
}

// rebuildWithProgress inserts -progress pipe:2 into the args before the final output path.
func rebuildWithProgress(args []string) []string {
	if len(args) == 0 {
		return args
	}
	output := args[len(args)-1]
	middle := args[:len(args)-1]
	result := make([]string, 0, len(middle)+3)
	result = append(result, middle...)
	result = append(result, "-progress", "pipe:2", output)
	return result
}

func (t *Transcoder) tempOutputPath(inputPath string) string {
	dir := t.tempDir
	if dir == "" {
		dir = filepath.Dir(inputPath)
	}
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, name+".pdarr-tmp"+ext)
}

// progressRegex matches ffmpeg progress lines like "time=00:01:23.45".
var progressRegex = regexp.MustCompile(`time=(\d+):(\d+):(\d+\.\d+)`)

// ParseFFmpegTime parses an ffmpeg time string "HH:MM:SS.ms" to seconds.
func ParseFFmpegTime(s string) float64 {
	m := progressRegex.FindStringSubmatch(s)
	if len(m) < 4 {
		return 0
	}
	h, _ := strconv.ParseFloat(m[1], 64)
	min, _ := strconv.ParseFloat(m[2], 64)
	sec, _ := strconv.ParseFloat(m[3], 64)
	return h*3600 + min*60 + sec
}

// FormatDuration formats a duration for logging.
func FormatDuration(d time.Duration) string {
	return d.Round(time.Second).String()
}
