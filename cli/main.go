package main

import (
	"os"

	rootcmd "github.com/dotechhq/zenith/cli/cmd/root"
)

func main() {
	if err := rootcmd.Execute(); err != nil {
		os.Exit(1)
	}
}
