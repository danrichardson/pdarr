package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// UpdateFile updates specific key=value pairs in a TOML config file in-place.
// All other content (comments, blank lines, other settings) is preserved.
// Inline comments on changed lines are not preserved.
// All keys in sqzarr.toml are globally unique across sections, so no section
// tracking is required.
func UpdateFile(path string, updates map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	content := string(data)
	for key, rawValue := range updates {
		re := regexp.MustCompile(`(?m)^(\s*` + regexp.QuoteMeta(key) + `\s*=\s*).*$`)
		if re.MatchString(content) {
			content = re.ReplaceAllString(content, "${1}"+rawValue)
		} else {
			// Key not present — this shouldn't happen for known settings,
			// but append it to a best-guess section as a safety net.
			content = strings.TrimRight(content, "\n") + "\n" + key + " = " + rawValue + "\n"
		}
	}

	return os.WriteFile(path, []byte(content), 0o644)
}
