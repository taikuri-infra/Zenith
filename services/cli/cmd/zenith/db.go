package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func dbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "db",
		Aliases: []string{"database", "databases"},
		Short:   "Manage databases",
	}

	cmd.AddCommand(
		dbListCmd(),
		dbCreateCmd(),
		dbConnectCmd(),
		dbDeleteCmd(),
	)
	return cmd
}

func dbListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all databases",
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			path := "/api/v1/databases"
			if cfg.ProjectID != "" {
				path += "?project_id=" + cfg.ProjectID
			}

			var result struct {
				Databases []struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Engine string `json:"engine"`
					Status string `json:"status"`
					SizeMB int    `json:"size_mb"`
				} `json:"databases"`
			}
			if err := client.Get(path, &result); err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tENGINE\tSTATUS\tSIZE")
			for _, db := range result.Databases {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%dMB\n", db.ID, db.Name, db.Engine, db.Status, db.SizeMB)
			}
			return w.Flush()
		},
	}
}

func dbCreateCmd() *cobra.Command {
	var engine string
	var sizeMB int

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			body := map[string]interface{}{
				"name":       args[0],
				"engine":     engine,
				"size_mb":    sizeMB,
				"project_id": cfg.ProjectID,
			}

			var result struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Host string `json:"host"`
			}
			if err := client.Post("/api/v1/databases", body, &result); err != nil {
				return err
			}

			fmt.Printf("Database created: %s (ID: %s)\n", result.Name, result.ID)
			if result.Host != "" {
				fmt.Printf("Host: %s\n", result.Host)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&engine, "engine", "e", "postgresql", "Database engine (postgresql, redis, mongodb)")
	cmd.Flags().IntVarP(&sizeMB, "size", "s", 500, "Storage size in MB")
	return cmd
}

func dbConnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connect <db-id>",
		Short: "Show database connection string",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			var result struct {
				ConnectionString string `json:"connection_string"`
				Host             string `json:"host"`
				Port             int    `json:"port"`
				Username         string `json:"username"`
				Database         string `json:"database"`
			}
			if err := client.Get("/api/v1/databases/"+args[0]+"/connection", &result); err != nil {
				return err
			}

			if result.ConnectionString != "" {
				fmt.Println(result.ConnectionString)
			} else {
				fmt.Printf("Host: %s\nPort: %d\nUser: %s\nDatabase: %s\n",
					result.Host, result.Port, result.Username, result.Database)
			}
			return nil
		},
	}
}

func dbDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <db-id>",
		Short: "Delete a database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			fmt.Printf("Are you sure you want to delete database %s? This cannot be undone. (y/N): ", args[0])
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "y" {
				fmt.Println("Cancelled.")
				return nil
			}

			if err := client.Delete("/api/v1/databases/" + args[0]); err != nil {
				return err
			}
			fmt.Println("Database deleted.")
			return nil
		},
	}
}
