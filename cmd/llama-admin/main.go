package main

import (
	"llama-admin/internal/cmd"
)

func main() {
	cmd.Version = "dev"
	cmd.Commit = "none"
	cmd.BuildTime = "unknown"
	cmd.Execute()
}
