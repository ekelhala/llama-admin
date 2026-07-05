package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var allowedEmailsCmd = &cobra.Command{
	Use:   "allowed-emails",
	Short: "Allowed email management commands",
}

var allowedEmailsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List allowed emails",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		data, err := c.Get("/api/v1/auth/allowed-emails")
		if err != nil {
			return err
		}

		var resp map[string]any
		json.Unmarshal(data, &resp)

		if outputFormat == "json" {
			PrintJSON(resp)
			return nil
		}

		emails, _ := resp["emails"].([]any)
		headers := []string{"EMAIL", "SOURCE"}
		var rows [][]string
		for _, e := range emails {
			m := e.(map[string]any)
			rows = append(rows, []string{
				m["email"].(string),
				m["source"].(string),
			})
		}
		PrintTable(headers, rows)
		return nil
	},
}

var allowedEmailsAddCmd = &cobra.Command{
	Use:   "add <email>",
	Short: "Add an email to the allowlist",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		_, err = c.Post("/api/v1/auth/allowed-emails", map[string]any{
			"email": args[0],
		})
		if err != nil {
			return err
		}

		PrintSuccess("Email added to allowlist")
		return nil
	},
}

var allowedEmailsRemoveCmd = &cobra.Command{
	Use:   "remove <email>",
	Short: "Remove an email from the allowlist",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		_, err = c.Delete("/api/v1/auth/allowed-emails/" + args[0])
		if err != nil {
			return err
		}

		PrintSuccess("Email removed from allowlist")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(allowedEmailsCmd)
	allowedEmailsCmd.AddCommand(allowedEmailsListCmd)
	allowedEmailsCmd.AddCommand(allowedEmailsAddCmd)
	allowedEmailsCmd.AddCommand(allowedEmailsRemoveCmd)
}
