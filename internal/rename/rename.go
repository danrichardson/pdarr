// Package rename computes codec-aware output filenames for transcoded files.
package rename

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// h.?264 family: h264, H264, h.264, H.264
	reh264 = regexp.MustCompile(`(?i)h(\.?)264`)
	// x.?264 family: x264, X264, x.264, X.264
	rex264 = regexp.MustCompile(`(?i)x(\.?)264`)
	// AVC (word-boundary only to avoid false positives in longer tokens)
	reavc = regexp.MustCompile(`(?i)\bAVC\b`)
)

// OutputName returns the new filename for a transcoded file.
// Codec tokens are replaced (h264→h265, x264→x265, AVC→HEVC) and the
// extension is always changed to .mkv. Directory component is not touched.
func OutputName(sourceName string) string {
	ext := filepath.Ext(sourceName)
	stem := sourceName[:len(sourceName)-len(ext)]
	stem = replaceCodecTokens(stem)
	return stem + ".mkv"
}

// OutputPath returns the full path for the transcoded file alongside the source.
// It avoids collisions with existing files by appending _1, _2, etc.
// Pass FileExists as checkExist in production; use a stub in tests.
func OutputPath(sourceDir, name string, checkExist func(string) bool) string {
	candidate := filepath.Join(sourceDir, name)
	if !checkExist(candidate) {
		return candidate
	}
	ext := filepath.Ext(name)
	base := name[:len(name)-len(ext)]
	for i := 1; i <= 99; i++ {
		candidate = filepath.Join(sourceDir, fmt.Sprintf("%s_%d%s", base, i, ext))
		if !checkExist(candidate) {
			return candidate
		}
	}
	return filepath.Join(sourceDir, fmt.Sprintf("%s_out%s", base, ext))
}

// FileExists is the production check-exist function for OutputPath.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// replaceCodecTokens substitutes codec version tokens in a filename stem.
func replaceCodecTokens(stem string) string {
	// h.?264 → h.?265, preserving case and the optional dot.
	stem = reh264.ReplaceAllStringFunc(stem, func(m string) string {
		dot := ""
		if strings.Contains(m, ".") {
			dot = "."
		}
		if m[0] == 'H' {
			return "H" + dot + "265"
		}
		return "h" + dot + "265"
	})

	// x.?264 → x.?265, preserving case and the optional dot.
	stem = rex264.ReplaceAllStringFunc(stem, func(m string) string {
		dot := ""
		if strings.Contains(m, ".") {
			dot = "."
		}
		if m[0] == 'X' {
			return "X" + dot + "265"
		}
		return "x" + dot + "265"
	})

	// AVC → HEVC, preserving case style of the match.
	stem = reavc.ReplaceAllStringFunc(stem, func(m string) string {
		upper := strings.ToUpper(m)
		lower := strings.ToLower(m)
		switch m {
		case upper:
			return "HEVC"
		case lower:
			return "hevc"
		default:
			return "Hevc"
		}
	})

	return stem
}
