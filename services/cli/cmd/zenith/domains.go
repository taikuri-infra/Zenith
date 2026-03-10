package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func domainsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "domains",
		Aliases: []string{"domain"},
		Short:   "Manage custom domains",
	}

	cmd.AddCommand(domainsListCmd(), domainsAddCmd())
	return cmd
}

func domainsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <app-id>",
		Short: "List custom domains for an app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			var result struct {
				Domains []struct {
					ID       string `json:"id"`
					Domain   string `json:"domain"`
					Status   string `json:"status"`
					TLSReady bool   `json:"tls_ready"`
				} `json:"domains"`
			}
			if err := client.Get("/api/v1/apps/"+args[0]+"/domains", &result); err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tDOMAIN\tSTATUS\tTLS")
			for _, d := range result.Domains {
				tls := "no"
				if d.TLSReady {
					tls = "yes"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.ID, d.Domain, d.Status, tls)
			}
			return w.Flush()
		},
	}
}

func domainsAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <app-id> <domain>",
		Short: "Add a custom domain to an app",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()

			body := map[string]string{"domain": args[1]}
			var result struct {
				ID     string `json:"id"`
				Domain string `json:"domain"`
			}
			if err := client.Post("/api/v1/apps/"+args[0]+"/domains", body, &result); err != nil {
				return err
			}

			fmt.Printf("Domain %s added (ID: %s)\n", result.Domain, result.ID)
			fmt.Println("Point your DNS CNAME to your app's subdomain.")
			return nil
		},
	}
}
