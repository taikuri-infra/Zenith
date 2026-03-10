package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dotechhq/zenith/services/cli/internal/config"
)

func loginCmd() *cobra.Command {
	var email, password, apiURL string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Zenith",
		Long:  "Log in to Zenith using email and password. Tokens are stored in ~/.zenith/config.json.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if apiURL != "" {
				cfg.APIBaseURL = strings.TrimRight(apiURL, "/")
			}

			if email == "" {
				fmt.Print("Email: ")
				fmt.Scanln(&email)
			}
			if password == "" {
				fmt.Print("Password: ")
				fmt.Scanln(&password)
			}

			if err := client.Login(email, password); err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			fmt.Println("Logged in successfully.")
			fmt.Printf("Config saved to %s\n", config.ConfigPath())
			return nil
		},
	}

	cmd.Flags().StringVarP(&email, "email", "e", "", "Account email")
	cmd.Flags().StringVarP(&password, "password", "p", "", "Account password")
	cmd.Flags().StringVar(&apiURL, "api-url", "", "Custom API URL (default: https://api.freezenith.com)")

	return cmd
}

func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and clear stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.AccessToken = ""
			cfg.RefreshToken = ""
			if err := cfg.Save(); err != nil {
				return err
			}
			if err := os.Remove(config.ConfigPath()); err != nil && !os.IsNotExist(err) {
				return err
			}
			fmt.Println("Logged out.")
			return nil
		},
	}
}
