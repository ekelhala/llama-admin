package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var instancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "Instance management commands",
}

var instancesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List instances",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		data, err := c.Get("/api/v1/instances")
		if err != nil {
			return err
		}

		var instances []map[string]any
		if err := json.Unmarshal(data, &instances); err != nil {
			return err
		}

		if outputFormat == "json" {
			PrintJSON(instances)
			return nil
		}

		headers := []string{"NAME", "STATUS", "PORT", "CREATED"}
		var rows [][]string
		for _, inst := range instances {
			rows = append(rows, []string{
				inst["name"].(string),
				inst["status"].(string),
				fmt.Sprintf("%v", inst["port"]),
				fmt.Sprintf("%v", inst["created"]),
			})
		}
		PrintTable(headers, rows)
		return nil
	},
}

var instancesGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get instance detail",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		data, err := c.Get("/api/v1/instances/" + args[0])
		if err != nil {
			return err
		}

		PrintJSON(data)
		return nil
	},
}

var instancesCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create an instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		name := args[0]
		model, _ := cmd.Flags().GetString("model")
		ctxSize, _ := cmd.Flags().GetInt("ctx-size")
		gpuLayers, _ := cmd.Flags().GetInt("gpu-layers")

		opts := map[string]any{
			"backend_type": "llama_cpp",
			"backend_options": map[string]any{
				"model": model,
			},
		}
		if ctxSize > 0 {
			opts["backend_options"].(map[string]any)["ctx_size"] = ctxSize
		}
		if gpuLayers > 0 {
			opts["backend_options"].(map[string]any)["n_gpu_layers"] = gpuLayers
		}

		data, err := c.Post("/api/v1/instances/"+name, opts)
		if err != nil {
			return err
		}

		PrintJSON(data)
		return nil
	},
}

var instancesStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start an instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		data, err := c.Post("/api/v1/instances/"+args[0]+"/start", nil)
		if err != nil {
			return err
		}

		PrintJSON(data)
		return nil
	},
}

var instancesStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop an instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		data, err := c.Post("/api/v1/instances/"+args[0]+"/stop", nil)
		if err != nil {
			return err
		}

		PrintJSON(data)
		return nil
	},
}

var instancesRestartCmd = &cobra.Command{
	Use:   "restart <name>",
	Short: "Restart an instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		data, err := c.Post("/api/v1/instances/"+args[0]+"/restart", nil)
		if err != nil {
			return err
		}

		PrintJSON(data)
		return nil
	},
}

var instancesDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		_, err = c.Delete("/api/v1/instances/" + args[0])
		if err != nil {
			return err
		}

		PrintSuccess("Instance deleted")
		return nil
	},
}

var instancesLogsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Get instance logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		lines := 200
		if l := cmd.Flag("lines"); l != nil {
			lines, _ = strconv.Atoi(l.Value.String())
		}

		data, err := c.Get("/api/v1/instances/" + args[0] + "/logs?lines=" + fmt.Sprintf("%d", lines))
		if err != nil {
			return err
		}

		var resp map[string]string
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		}

		fmt.Print(resp["logs"])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(instancesCmd)
	instancesCmd.AddCommand(instancesListCmd)
	instancesCmd.AddCommand(instancesGetCmd)
	instancesCmd.AddCommand(instancesCreateCmd)
	instancesCmd.AddCommand(instancesStartCmd)
	instancesCmd.AddCommand(instancesStopCmd)
	instancesCmd.AddCommand(instancesRestartCmd)
	instancesCmd.AddCommand(instancesDeleteCmd)
	instancesCmd.AddCommand(instancesLogsCmd)

	instancesCreateCmd.Flags().String("model", "", "model path")
	instancesCreateCmd.Flags().Int("ctx-size", 0, "context size")
	instancesCreateCmd.Flags().Int("gpu-layers", 0, "number of GPU layers")
	instancesLogsCmd.Flags().IntP("lines", "n", 200, "number of log lines")
}
