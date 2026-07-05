package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var apiKeysCmd = &cobra.Command{
	Use:   "api-keys",
	Short: "API key management commands",
}

var apiKeysCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create an API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		mode, _ := cmd.Flags().GetString("mode")
		expires, _ := cmd.Flags().GetString("expires")

		req := map[string]any{
			"name":             args[0],
			"permission_mode":  mode,
		}
		if expires != "" {
			d, _ := time.ParseDuration(expires)
			req["expires_at"] = int64(time.Now().Add(d).Unix())
		}

		data, err := c.Post("/api/v1/auth/keys", req)
		if err != nil {
			return err
		}

		var key map[string]any
		json.Unmarshal(data, &key)
		fmt.Printf("API key created! Save this key - it won't be shown again:\n\n  %s\n\n", key["key"])
		return nil
	},
}

var apiKeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		data, err := c.Get("/api/v1/auth/keys")
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			PrintJSON(data)
			return nil
		}

		var keys []map[string]any
		json.Unmarshal(data, &keys)

		headers := []string{"ID", "NAME", "MODE", "CREATED"}
		var rows [][]string
		for _, k := range keys {
			rows = append(rows, []string{
				fmt.Sprintf("%v", k["id"]),
				k["name"].(string),
				k["permission_mode"].(string),
				fmt.Sprintf("%v", k["created_at"]),
			})
		}
		PrintTable(headers, rows)
		return nil
	},
}

var apiKeysDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		_, err = c.Delete("/api/v1/auth/keys/" + args[0])
		if err != nil {
			return err
		}

		PrintSuccess("API key deleted")
		return nil
	},
}

var apiKeysGrantCmd = &cobra.Command{
	Use:   "grant <key_id> --instance <name>",
	Short: "Grant instance permission to an API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		instanceName, _ := cmd.Flags().GetString("instance")
		if instanceName == "" {
			return fmt.Errorf("--instance is required")
		}

		instData, err := c.Get("/api/v1/instances/" + instanceName)
		if err != nil {
			return err
		}
		var inst map[string]any
		json.Unmarshal(instData, &inst)
		instanceID := fmt.Sprintf("%v", inst["id"])

		_, err = c.Post("/api/v1/auth/keys/"+args[0]+"/permissions", map[string]any{
			"instance_id": instanceID,
		})
		if err != nil {
			return err
		}

		PrintSuccess("Permission granted")
		return nil
	},
}

var apiKeysRevokeCmd = &cobra.Command{
	Use:   "revoke <key_id> --instance <name>",
	Short: "Revoke instance permission from an API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		instanceName, _ := cmd.Flags().GetString("instance")
		if instanceName == "" {
			return fmt.Errorf("--instance is required")
		}

		instData, err := c.Get("/api/v1/instances/" + instanceName)
		if err != nil {
			return err
		}
		var inst map[string]any
		json.Unmarshal(instData, &inst)
		instanceID := fmt.Sprintf("%v", inst["id"])

		_, err = c.Delete("/api/v1/auth/keys/" + args[0] + "/permissions/" + instanceID)
		if err != nil {
			return err
		}

		PrintSuccess("Permission revoked")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(apiKeysCmd)
	apiKeysCmd.AddCommand(apiKeysCreateCmd)
	apiKeysCmd.AddCommand(apiKeysListCmd)
	apiKeysCmd.AddCommand(apiKeysDeleteCmd)
	apiKeysCmd.AddCommand(apiKeysGrantCmd)
	apiKeysCmd.AddCommand(apiKeysRevokeCmd)

	apiKeysCreateCmd.Flags().String("mode", "per_instance", "permission mode: allow_all or per_instance")
	apiKeysCreateCmd.Flags().String("expires", "", "expiration duration (e.g. 24h, 7d)")
	apiKeysGrantCmd.Flags().String("instance", "", "instance name")
	apiKeysRevokeCmd.Flags().String("instance", "", "instance name")
}
