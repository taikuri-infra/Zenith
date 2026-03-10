package main

import (
	"fmt"
	"text/tabwriter"
	"os"

	"github.com/spf13/cobra"
)

func projectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project",
		Aliases: []string{"projects"},
		Short:   "Manage projects",
	}

	cmd.AddCommand(projectListCmd(), projectSelectCmd())
	return cmd
}

func projectListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			var result struct {
				Projects []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"projects"`
			}
			if err := client.Get("/api/v1/projects", &result); err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME")
			for _, p := range result.Projects {
				active := ""
				if p.ID == cfg.ProjectID {
					active = " (active)"
				}
				fmt.Fprintf(w, "%s\t%s%s\n", p.ID, p.Name, active)
			}
			return w.Flush()
		},
	}
}

func projectSelectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "select <project-id>",
		Short: "Set the active project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()
			cfg.ProjectID = args[0]
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Printf("Active project set to %s\n", args[0])
			return nil
		},
	}
}
