package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func storageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "Manage storage buckets",
	}

	cmd.AddCommand(
		storageListCmd(),
		storageCreateCmd(),
	)
	return cmd
}

func storageListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List storage buckets",
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			path := "/api/v1/storage"
			if cfg.ProjectID != "" {
				path += "?project_id=" + cfg.ProjectID
			}

			var result struct {
				Buckets []struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Access string `json:"access"`
					SizeMB int64  `json:"size_mb"`
				} `json:"buckets"`
			}
			if err := client.Get(path, &result); err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tACCESS\tSIZE")
			for _, b := range result.Buckets {
				fmt.Fprintf(w, "%s\t%s\t%s\t%dMB\n", b.ID, b.Name, b.Access, b.SizeMB)
			}
			return w.Flush()
		},
	}
}

func storageCreateCmd() *cobra.Command {
	var access string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a storage bucket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			body := map[string]string{
				"name":       args[0],
				"access":     access,
				"project_id": cfg.ProjectID,
			}

			var result struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			if err := client.Post("/api/v1/storage", body, &result); err != nil {
				return err
			}

			fmt.Printf("Bucket created: %s (ID: %s)\n", result.Name, result.ID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&access, "access", "a", "private", "Access level (private, public)")
	return cmd
}
