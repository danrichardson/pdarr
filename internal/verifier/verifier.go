package verifier

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
)

// Result holds the outcome of an output file verification.
type Result struct {
	InputSize    int64
	OutputSize   int64
	InputDur     float64
	OutputDur    float64
	DurationDiff float64
	OK           bool
	Reason       string
}

// Verify checks that outputPath is valid: smaller than inputPath and
// duration matches within toleranceSecs.
func Verify(inputPath, outputPath string, toleranceSecs float64) (*Result, error) {
	inputInfo, err := os.Stat(inputPath)
	if err != nil {
		return nil, fmt.Errorf("stat input: %w", err)
	}
	outputInfo, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("stat output: %w", err)
	}

	r := &Result{
		InputSize:  inputInfo.Size(),
		OutputSize: outputInfo.Size(),
	}

	// Probe output duration.
	outputDur, err := probeDuration(outputPath)
	if err != nil {
		return nil, fmt.Errorf("probe output: %w", err)
	}
	inputDur, err := probeDuration(inputPath)
	if err != nil {
		return nil, fmt.Errorf("probe input: %w", err)
	}

	r.InputDur = inputDur
	r.OutputDur = outputDur
	r.DurationDiff = math.Abs(outputDur - inputDur)

	if r.OutputSize >= r.InputSize {
		r.Reason = fmt.Sprintf("output (%d bytes) is not smaller than input (%d bytes)", r.OutputSize, r.InputSize)
		return r, nil
	}
	if r.DurationDiff > toleranceSecs {
		r.Reason = fmt.Sprintf("duration mismatch: input %.2fs, output %.2fs (diff %.2fs > tolerance %.2fs)",
			inputDur, outputDur, r.DurationDiff, toleranceSecs)
		return r, nil
	}

	r.OK = true
	return r, nil
}

type ffprobeFormat struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func probeDuration(path string) (float64, error) {
	out, err := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		path,
	).Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w", err)
	}

	var result ffprobeFormat
	if err := json.Unmarshal(out, &result); err != nil {
		return 0, fmt.Errorf("parse ffprobe output: %w", err)
	}

	var dur float64
	fmt.Sscanf(result.Format.Duration, "%f", &dur)
	return dur, nil
}
