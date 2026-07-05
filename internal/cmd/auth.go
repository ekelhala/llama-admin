package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"llama-admin/internal/auth"
	innerclient "llama-admin/internal/client"
	innerconfig "llama-admin/internal/config"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var loginCmd = &cobra.Command{
	Use:   "login [provider]",
	Short: "Login via OAuth device flow",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := innerconfig.Load(cfgPath)
		if err != nil {
			return err
		}

		provider := ""
		if len(args) > 0 {
			provider = args[0]
		}
		if provider == "" {
			provider = cfg.Provider
		}

		c := innerclient.New(cfg.ServerURL, "")
		result, err := auth.CompleteDeviceFlow(c, provider)
		if err != nil {
			return err
		}

		cfg.SessionToken = result.SessionToken
		cfg.Provider = provider
		if serverURL != "" {
			cfg.ServerURL = serverURL
		}
		if err := innerconfig.Save(cfgPath, cfg); err != nil {
			return err
		}

		userData, _ := json.MarshalIndent(result.User, "", "  ")
		fmt.Printf("Login successful!\nUser: %s\n%s\n", result.User["username"], string(userData))
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current auth status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := innerconfig.Load(cfgPath)
		if err != nil {
			return err
		}

		if cfg.SessionToken == "" {
			fmt.Println("Not logged in. Run `llama-admin auth login`")
			return nil
		}

		c := innerclient.New(cfg.ServerURL, cfg.SessionToken)
		data, err := c.Get("/api/v1/auth/session")
		if err != nil {
			return err
		}

		fmt.Printf("Server: %s\nProvider: %s\n", cfg.ServerURL, cfg.Provider)
		if outputFormat == "json" {
			PrintJSON(map[string]any{
				"server":   cfg.ServerURL,
				"provider": cfg.Provider,
				"session":  data,
			})
		} else {
			fmt.Printf("Session: %s...\n", cfg.SessionToken[:10])
		}
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and clear local state",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := innerconfig.Load(cfgPath)
		if err != nil {
			return err
		}

		if cfg.SessionToken == "" {
			fmt.Println("Not logged in")
			return nil
		}

		c := innerclient.New(cfg.ServerURL, cfg.SessionToken)
		_, err = c.Delete("/api/v1/auth/session")
		if err != nil {
			fmt.Printf("Warning: failed to revoke session on server: %v\n", err)
		}

		cfg.SessionToken = ""
		cfg.Provider = ""
		if err := innerconfig.Save(cfgPath, cfg); err != nil {
			return err
		}

		fmt.Println("Logged out successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(logoutCmd)
}
