package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Model management commands",
}

var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available models",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		data, err := c.Get("/api/v1/models")
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			PrintJSONBytes(data)
			return nil
		}

		var models []map[string]any
		if err := json.Unmarshal(data, &models); err != nil {
			return err
		}

		headers := []string{"ALIAS", "NAME", "SOURCE", "SIZE", "PATH"}
		var rows [][]string
		for _, m := range models {
			sizeBytes, _ := toInt64(m["size_bytes"])
			rows = append(rows, []string{
				fmt.Sprintf("%v", m["alias"]),
				fmt.Sprintf("%v", m["name"]),
				fmt.Sprintf("%v", m["source"]),
				FormatBytes(sizeBytes),
				fmt.Sprintf("%v", m["path"]),
			})
		}
		PrintTable(headers, rows)
		return nil
	},
}

var modelsRegisterCmd = &cobra.Command{
	Use:   "register <alias> <filename>",
	Short: "Register a model alias -> filename mapping",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		data, err := c.Post("/api/v1/models", map[string]any{
			"alias":    args[0],
			"filename": args[1],
		})
		if err != nil {
			return err
		}

		PrintJSONBytes(data)
		return nil
	},
}

var modelsGetCmd = &cobra.Command{
	Use:   "get <alias>",
	Short: "Get a model by alias",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		data, err := c.Get("/api/v1/models/" + args[0])
		if err != nil {
			return err
		}

		PrintJSONBytes(data)
		return nil
	},
}

var modelsDeleteCmd = &cobra.Command{
	Use:   "delete <alias>",
	Short: "Delete a model by alias",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		_, err = c.Delete("/api/v1/models/" + args[0])
		if err != nil {
			return err
		}

		PrintSuccess("Model deleted")
		return nil
	},
}

var modelsFilesCmd = &cobra.Command{
	Use:   "files",
	Short: "List raw disk-scanned GGUF files",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		data, err := c.Get("/api/v1/models/files")
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			PrintJSONBytes(data)
			return nil
		}

		var files []map[string]any
		if err := json.Unmarshal(data, &files); err != nil {
			return err
		}

		headers := []string{"FILENAME", "SIZE"}
		var rows [][]string
		for _, f := range files {
			sizeBytes, _ := toInt64(f["size_bytes"])
			rows = append(rows, []string{
				fmt.Sprintf("%v", f["filename"]),
				FormatBytes(sizeBytes),
			})
		}
		PrintTable(headers, rows)
		return nil
	},
}

var modelsDownloadCmd = &cobra.Command{
	Use:   "download <repo_id>",
	Short: "Download a model from Hugging Face Hub",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		filename, _ := cmd.Flags().GetString("filename")
		revision, _ := cmd.Flags().GetString("revision")

		if filename == "" {
			return fmt.Errorf("--filename is required")
		}

		req := map[string]any{
			"repo_id":  args[0],
			"filename": filename,
		}
		if revision != "" {
			req["revision"] = revision
		}

		data, err := c.Post("/api/v1/models/download", req)
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			PrintJSONBytes(data)
			return nil
		}

		var resp map[string]any
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		}

		fmt.Printf("Download started (job_id: %s, status: %s)\n", resp["job_id"], resp["status"])
		fmt.Println("Use `llama-admin models jobs get <job_id>` to track progress.")
		return nil
	},
}

var modelsJobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Download job management commands",
}

var modelsJobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List download jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		data, err := c.Get("/api/v1/models/download/jobs")
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			PrintJSONBytes(data)
			return nil
		}

		var jobs []map[string]any
		if err := json.Unmarshal(data, &jobs); err != nil {
			return err
		}

		headers := []string{"JOB_ID", "REPO_ID", "FILENAME", "STATUS", "PROGRESS", "ERROR"}
		var rows [][]string
		for _, j := range jobs {
			progress, _ := j["progress"].(map[string]any)
			errStr, _ := j["error"].(string)
			rows = append(rows, []string{
				fmt.Sprintf("%v", j["job_id"]),
				fmt.Sprintf("%v", j["repo_id"]),
				fmt.Sprintf("%v", j["filename"]),
				fmt.Sprintf("%v", j["status"]),
				FormatProgress(progress),
				errStr,
			})
		}
		PrintTable(headers, rows)
		return nil
	},
}

var modelsJobsGetCmd = &cobra.Command{
	Use:   "get <job_id>",
	Short: "Get download job status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		data, err := c.Get("/api/v1/models/download/jobs/" + args[0])
		if err != nil {
			return err
		}

		PrintJSONBytes(data)
		return nil
	},
}

var modelsJobsCancelCmd = &cobra.Command{
	Use:   "cancel <job_id>",
	Short: "Cancel a download job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := GetClient()
		if err != nil {
			return err
		}

		if _, err := c.Delete("/api/v1/models/download/jobs/" + args[0]); err != nil {
			return err
		}

		PrintSuccess("Download job cancelled")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(modelsCmd)
	modelsCmd.AddCommand(modelsListCmd)
	modelsCmd.AddCommand(modelsFilesCmd)
	modelsCmd.AddCommand(modelsDownloadCmd)
	modelsCmd.AddCommand(modelsRegisterCmd)
	modelsCmd.AddCommand(modelsGetCmd)
	modelsCmd.AddCommand(modelsDeleteCmd)
	modelsCmd.AddCommand(modelsJobsCmd)
	modelsJobsCmd.AddCommand(modelsJobsListCmd)
	modelsJobsCmd.AddCommand(modelsJobsGetCmd)
	modelsJobsCmd.AddCommand(modelsJobsCancelCmd)

	modelsDownloadCmd.Flags().String("filename", "", "filename to download (e.g. model.gguf)")
	modelsDownloadCmd.Flags().String("revision", "", "revision/branch (default: main)")
}
