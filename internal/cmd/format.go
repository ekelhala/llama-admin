package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

func PrintJSON(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

func PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
}

func PrintSuccess(msg string) {
	fmt.Println(msg)
}

// FormatBytes renders a byte count as a human-readable string (e.g. "1.23 GB").
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffix := "KMGTPE"
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), suffix[exp])
}

// FormatProgress renders a download progress map as "downloaded / total (pct%)".
// Returns "-" when the total size is unknown.
func FormatProgress(progress map[string]any) string {
	if progress == nil {
		return "-"
	}
	downloaded, _ := toInt64(progress["bytes_downloaded"])
	total, _ := toInt64(progress["bytes_total"])
	percent, _ := toFloat64(progress["percent"])

	if total <= 0 {
		if downloaded <= 0 {
			return "-"
		}
		return FormatBytes(downloaded)
	}
	return fmt.Sprintf("%s / %s (%.1f%%)", FormatBytes(downloaded), FormatBytes(total), percent)
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	}
	return 0, false
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	}
	return 0, false
}
