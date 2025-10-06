package main

import (
	"fmt"
	"os"

	"github.com/phillarmonic/drun/cmd/drun/app"
)

// Version information (set at build time)
var (
	version = "2.0.0-dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Create and execute CLI application
	cliApp := app.NewApp(version, commit, date)

	if err := cliApp.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
