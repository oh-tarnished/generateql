// Package cmd implements the generateql command-line interface.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command. Subcommands (introspect, generate) are registered
// from their own files via init().
var rootCmd = &cobra.Command{
	Use:   "generateql",
	Short: "Generate typed GraphQL clients from a live endpoint",
	Long: `generateql introspects a GraphQL endpoint and generates a fully typed client
— models plus query, mutation, and subscription functions per resource — that runs on
top of the oh-tarnished network runtime.`,
}

// Execute runs the root command. It is called by main and exits non-zero on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
