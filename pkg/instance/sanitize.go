package instance

import (
	"fmt"
	"regexp"
	"sort"
)

var paramKeyRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

var blockedParams = map[string]struct{}{
	"model": {},
	"host":  {},
	"port":  {},
}

// SanitizeParams validates user-supplied llama-server parameters.
// Keys must match ^[a-z][a-z0-9-]*$ and not be in the blocked set
// (model, host, port — those are managed by llama-admin itself).
// Returns a sorted, deterministic slice of "key=value" or "key" pairs.
// An empty value emits a bare flag (e.g. "flash-attn" with value "" → "--flash-attn").
func SanitizeParams(params map[string]string) ([]string, error) {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	args := make([]string, 0, len(params))
	for _, k := range keys {
		if _, blocked := blockedParams[k]; blocked {
			return nil, fmt.Errorf("parameter %q is not allowed (managed by llama-admin)", k)
		}
		if !paramKeyRegex.MatchString(k) {
			return nil, fmt.Errorf("invalid parameter key %q: must match ^[a-z][a-z0-9-]*$", k)
		}
		v := params[k]
		if v == "" {
			args = append(args, "--"+k)
		} else {
			args = append(args, "--"+k+"="+v)
		}
	}
	return args, nil
}
