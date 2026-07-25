package main

import (
	"fmt"
	"os"

	rootcmd "github.com/dotechhq/zenith/cli/cmd/root"
)

func main() {
	// The root command sets SilenceErrors so cobra doesn't print usage on every
	// failure; surface the error here so users actually see what went wrong.
	if err := rootcmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
