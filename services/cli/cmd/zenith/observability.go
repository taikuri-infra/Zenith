package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func logsCmd() *cobra.Command {
	var level, search, since string
	var limit int

	cmd := &cobra.Command{
		Use:   "logs <app-id>",
		Short: "View application logs from Loki",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()
			appID := args[0]

			path := fmt.Sprintf("/api/v1/apps/%s/logs?limit=%d&since=%s", appID, limit, since)
			if level != "" {
				path += "&level=" + level
			}
			if search != "" {
				path += "&search=" + search
			}

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
				fmt.Printf("%s [%-5s] %s\n", ts, e.Level, e.Line)
			}

			if len(result.Entries) == 0 {
				fmt.Println("No log entries found.")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&level, "level", "l", "", "Filter by log level (info, warn, error)")
	cmd.Flags().StringVarP(&search, "search", "s", "", "Search string filter")
	cmd.Flags().StringVar(&since, "since", "1h", "Time range (1h, 6h, 24h, 7d)")
	cmd.Flags().IntVarP(&limit, "limit", "n", 100, "Maximum number of entries")

	return cmd
}

func metricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics <app-id>",
		Short: "View application metrics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireAuth()
			appID := args[0]

			var overview struct {
				CPUPercent  float64 `json:"cpu_percent"`
				MemoryMB   float64 `json:"memory_mb"`
				MemoryPct  float64 `json:"memory_percent"`
				ReqRate    float64 `json:"request_rate"`
				ErrorRate  float64 `json:"error_rate"`
				P95Latency float64 `json:"p95_latency_ms"`
				PodCount   int     `json:"pod_count"`
			}
			if err := client.Get("/api/v1/apps/"+appID+"/metrics/overview", &overview); err != nil {
				return err
			}

			fmt.Printf("CPU:        %.1f%%\n", overview.CPUPercent)
			fmt.Printf("Memory:     %.0f MB (%.1f%%)\n", overview.MemoryMB, overview.MemoryPct)
			fmt.Printf("Requests:   %.1f/s\n", overview.ReqRate)
			fmt.Printf("Error Rate: %.2f%%\n", overview.ErrorRate)
			fmt.Printf("P95 Lat:    %.0f ms\n", overview.P95Latency)
			fmt.Printf("Pods:       %d\n", overview.PodCount)

			return nil
		},
	}

	return cmd
}
