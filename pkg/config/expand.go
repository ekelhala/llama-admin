package config

import (
	"os"
	"regexp"
)

var placeholderRe = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-([^}]*))?\}`)

func ExpandPlaceholders(s string) string {
	return placeholderRe.ReplaceAllStringFunc(s, func(match string) string {
		vars := placeholderRe.FindStringSubmatch(match)
		if len(vars) < 2 {
			return match
		}
		name := vars[1]
		value := os.Getenv(name)
		if value != "" {
			return value
		}
		if len(vars) > 3 && vars[3] != "" {
			return vars[3]
		}
		return ""
	})
}
