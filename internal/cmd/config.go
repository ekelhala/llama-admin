package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	innerconfig "llama-admin/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Local config management commands",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := innerconfig.Load(cfgPath)
		if err != nil {
			return err
		}

		key, value := args[0], args[1]
		switch key {
		case "server_url":
			cfg.ServerURL = value
		case "provider":
			cfg.Provider = value
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}

		if err := innerconfig.Save(cfgPath, cfg); err != nil {
			return err
		}

		PrintSuccess(fmt.Sprintf("Config %s set to %s", key, value))
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := innerconfig.Load(cfgPath)
		if err != nil {
			return err
		}

		PrintJSON(cfg)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
}
