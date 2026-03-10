package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func appsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apps",
		Aliases: []string{"app"},
		Short:   "Manage applications",
	}

	cmd.AddCommand(
		appsListCmd(),
		appsCreateCmd(),
		appsDeployCmd(),
		appsLogsCmd(),
		appsEnvCmd(),
		appsScaleCmd(),
		appsDeleteCmd(),
	)
	return cmd
}

func appsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all apps",
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			path := "/api/v1/apps"
			if cfg.ProjectID != "" {
				path += "?project_id=" + cfg.ProjectID
			}

			var result struct {
				Items []struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Status string `json:"status"`
					URL    string `json:"url"`
				} `json:"items"`
			}
			if err := client.Get(path, &result); err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tSTATUS\tURL")
			for _, a := range result.Items {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.ID, a.Name, a.Status, a.URL)
			}
			return w.Flush()
		},
	}
}

func appsCreateCmd() *cobra.Command {
	var gitRepo, branch string
	var port int

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			body := map[string]interface{}{
				"name":       args[0],
				"project_id": cfg.ProjectID,
			}
			if gitRepo != "" {
				body["git_repo"] = gitRepo
			}
			if branch != "" {
				body["branch"] = branch
			}
			if port > 0 {
				body["port"] = port
			}

			var result struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			if err := client.Post("/api/v1/apps", body, &result); err != nil {
				return err
			}

			fmt.Printf("App created: %s (ID: %s)\n", result.Name, result.ID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&gitRepo, "git-repo", "g", "", "Git repository URL")
	cmd.Flags().StringVarP(&branch, "branch", "b", "main", "Git branch")
	cmd.Flags().IntVarP(&port, "port", "P", 0, "Container port")

	return cmd
}

func appsDeployCmd() *cobra.Command {
	var image string

	cmd := &cobra.Command{
		Use:   "deploy <app-id>",
		Short: "Deploy an app (trigger build or deploy image)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			body := map[string]string{}
			if image != "" {
				body["image"] = image
			}

			var result struct {
				DeploymentID string `json:"deployment_id"`
				Status       string `json:"status"`
			}
			if err := client.Post("/api/v1/apps/"+args[0]+"/deploy", body, &result); err != nil {
				return err
			}

			fmt.Printf("Deployment started: %s (status: %s)\n", result.DeploymentID, result.Status)
			return nil
		},
	}

	cmd.Flags().StringVarP(&image, "image", "i", "", "Docker image to deploy")
	return cmd
}

func appsLogsCmd() *cobra.Command {
	var follow bool
	var tail int

	cmd := &cobra.Command{
		Use:   "logs <app-id>",
		Short: "View app logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()
			appID := args[0]

			if follow {
				return streamLogs(appID)
			}

			path := fmt.Sprintf("/api/v1/apps/%s/logs?limit=%d", appID, tail)
			var result struct {
				Entries []struct {
					Timestamp string `json:"timestamp"`
					Level     string `json:"level"`
					Line      string `json:"line"`
				} `json:"entries"`
			}
			if err := client.Get(path, &result); err != nil {
				return err
			}

			for _, e := range result.Entries {
				ts := e.Timestamp
				if len(ts) > 19 {
					ts = ts[:19]
				}
				fmt.Printf("%s [%s] %s\n", ts, e.Level, e.Line)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Stream logs in real-time")
	cmd.Flags().IntVarP(&tail, "tail", "t", 50, "Number of log lines to show")
	return cmd
}

func streamLogs(appID string) error {
	url := cfg.APIBaseURL + "/api/v1/apps/" + appID + "/logs/stream?token=" + cfg.AccessToken
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var entry struct {
			Timestamp string `json:"timestamp"`
			Level     string `json:"level"`
			Line      string `json:"line"`
		}
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			continue
		}
		ts := entry.Timestamp
		if len(ts) > 19 {
			ts = ts[:19]
		}
		fmt.Printf("%s [%s] %s\n", ts, entry.Level, entry.Line)
	}
	return scanner.Err()
}

func appsEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage app environment variables",
	}

	cmd.AddCommand(appsEnvListCmd(), appsEnvSetCmd(), appsEnvDeleteCmd())
	return cmd
}

func appsEnvListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <app-id>",
		Short: "List environment variables",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			var result struct {
				Vars []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"env_vars"`
			}
			if err := client.Get("/api/v1/apps/"+args[0]+"/env", &result); err != nil {
				return err
			}

			for _, v := range result.Vars {
				fmt.Printf("%s=%s\n", v.Key, v.Value)
			}
			return nil
		},
	}
}

func appsEnvSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <app-id> KEY=VALUE [KEY=VALUE...]",
		Short: "Set environment variables",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()
			appID := args[0]

			vars := make(map[string]string)
			for _, kv := range args[1:] {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid format: %s (expected KEY=VALUE)", kv)
				}
				vars[parts[0]] = parts[1]
			}

			if err := client.Post("/api/v1/apps/"+appID+"/env", map[string]interface{}{"vars": vars}, nil); err != nil {
				return err
			}

			fmt.Printf("Set %d environment variable(s)\n", len(vars))
			return nil
		},
	}
}

func appsEnvDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <app-id> <KEY>",
		Short: "Delete an environment variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()
			return client.Delete("/api/v1/apps/" + args[0] + "/env/" + args[1])
		},
	}
}

func appsScaleCmd() *cobra.Command {
	var replicas int

	cmd := &cobra.Command{
		Use:   "scale <app-id>",
		Short: "Scale an app's replica count",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			body := map[string]int{"replicas": replicas}
			if err := client.Put("/api/v1/apps/"+args[0]+"/scale", body, nil); err != nil {
				return err
			}

			fmt.Printf("Scaled to %d replica(s)\n", replicas)
			return nil
		},
	}

	cmd.Flags().IntVarP(&replicas, "replicas", "r", 1, "Number of replicas")
	return cmd
}

func appsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <app-id>",
		Short: "Delete an app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			fmt.Printf("Are you sure you want to delete app %s? (y/N): ", args[0])
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "y" {
				fmt.Println("Cancelled.")
				return nil
			}

			if err := client.Delete("/api/v1/apps/" + args[0]); err != nil {
				return err
			}
			fmt.Println("App deleted.")
			return nil
		},
	}
}
