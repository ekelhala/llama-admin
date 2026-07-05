package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	innerclient "llama-admin/internal/client"
	innerconfig "llama-admin/internal/config"
)

var (
	Version   string
	Commit    string
	BuildTime string
)

var rootCmd = &cobra.Command{
	Use:   "llama-admin",
	Short: "llama-admin CLI",
}

var (
	cfgPath      string
	serverURL    string
	outputFormat string
)

func Execute() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("llama-admin %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
		os.Exit(0)
	}
	rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", innerconfig.DefaultPath(), "config file path")
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "", "server URL (overrides config)")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "output format: table, json")
}

func GetClient() (*innerclient.Client, error) {
	cfg, err := innerconfig.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	url := cfg.ServerURL
	if serverURL != "" {
		url = serverURL
	}

	token := cfg.SessionToken
	if token == "" {
		return nil, fmt.Errorf("not logged in. Run `llama-admin auth login`")
	}

	return innerclient.New(url, token), nil
}

func GetProvider() string {
	cfg, _ := innerconfig.Load(cfgPath)
	if serverURL == "" {
		serverURL = cfg.ServerURL
	}
	return cfg.Provider
}
