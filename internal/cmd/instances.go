package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	innerclient "llama-admin/internal/client"
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

		var inst map[string]any
		if err := json.Unmarshal(data, &inst); err != nil {
			PrintJSONBytes(data)
			return nil
		}

		if outputFormat == "json" {
			PrintJSON(inst)
			return nil
		}

		printInstanceDetail(inst)
		return nil
	},
}

func printInstanceDetail(inst map[string]any) {
	rows := [][]string{
		{"ID", fmt.Sprintf("%v", inst["id"])},
		{"Name", fmt.Sprintf("%v", inst["name"])},
		{"Status", fmt.Sprintf("%v", inst["status"])},
		{"Host", fmt.Sprintf("%v", inst["host"])},
		{"Port", fmt.Sprintf("%v", inst["port"])},
		{"PID", fmt.Sprintf("%v", inst["pid"])},
		{"Created", fmt.Sprintf("%v", inst["created"])},
		{"Updated", fmt.Sprintf("%v", inst["updated"])},
	}
	if owner, ok := inst["owner_user_id"]; ok {
		rows = append(rows, []string{"Owner", fmt.Sprintf("%v", owner)})
	}

	PrintTable([]string{"FIELD", "VALUE"}, rows)

	if opts, ok := inst["options"].(map[string]any); ok && len(opts) > 0 {
		fmt.Println()
		fmt.Println("Backend options:")
		optRows := [][]string{}
		for k, v := range opts {
			optRows = append(optRows, []string{k, fmt.Sprintf("%v", v)})
		}
		PrintTable([]string{"OPTION", "VALUE"}, optRows)
	}
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

		// Resolve the model argument against the catalog of downloaded
		// models. A bare alias (e.g. "Qwen3.5-9B-Q4_K_M"), the relative
		// model name, or the bare filename will all resolve to the full
		// on-disk path. Anything that does not match is passed through
		// unchanged so absolute paths still work.
		resolved, err := resolveModelArg(c, model)
		if err != nil {
			return err
		}
		if resolved != "" {
			model = resolved
		}

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

		PrintJSONBytes(data)
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

		PrintJSONBytes(data)

		if wait, _ := cmd.Flags().GetBool("wait"); wait {
			if err := waitForInstanceRunning(c, args[0], waitTimeout(cmd)); err != nil {
				return err
			}
		}
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

		PrintJSONBytes(data)
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

		PrintJSONBytes(data)

		if wait, _ := cmd.Flags().GetBool("wait"); wait {
			if err := waitForInstanceRunning(c, args[0], waitTimeout(cmd)); err != nil {
				return err
			}
		}
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

	instancesCreateCmd.Flags().String("model", "", "model alias, name, filename, or absolute path (resolved against the model catalog)")
	instancesCreateCmd.Flags().Int("ctx-size", 0, "context size")
	instancesCreateCmd.Flags().Int("gpu-layers", 0, "number of GPU layers")
	instancesLogsCmd.Flags().IntP("lines", "n", 200, "number of log lines")

	// The server starts instances asynchronously and returns 202 Accepted.
	// --wait makes the CLI poll the instance status until it is running
	// (or fails) before returning.
	instancesStartCmd.Flags().Bool("wait", false, "wait for the instance to become healthy before returning")
	instancesStartCmd.Flags().Duration("timeout", 10*time.Minute, "maximum time to wait when --wait is set")
	instancesRestartCmd.Flags().Bool("wait", false, "wait for the instance to become healthy before returning")
	instancesRestartCmd.Flags().Duration("timeout", 10*time.Minute, "maximum time to wait when --wait is set")
}

// waitTimeout resolves the --timeout flag value for the given command.
func waitTimeout(cmd *cobra.Command) time.Duration {
	if t, _ := cmd.Flags().GetDuration("timeout"); t > 0 {
		return t
	}
	return 10 * time.Minute
}

// waitForInstanceRunning polls the instance status until it reaches a terminal
// state (running or failed) or the timeout elapses. It mirrors the server-side
// health wait: the API returns 202 Accepted immediately and the instance
// transitions through "restarting" before becoming "running".
func waitForInstanceRunning(c *innerclient.Client, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		data, err := c.Get("/api/v1/instances/" + name)
		if err != nil {
			return err
		}

		var inst map[string]any
		if err := json.Unmarshal(data, &inst); err != nil {
			return err
		}

		status, _ := inst["status"].(string)
		switch status {
		case "running":
			PrintSuccess("Instance " + name + " is running")
			return nil
		case "failed":
			return fmt.Errorf("instance %s failed to start", name)
		case "stopped":
			return fmt.Errorf("instance %s stopped", name)
		}

		time.Sleep(interval)
	}

	return fmt.Errorf("timed out waiting for instance %s to become healthy", name)
}
