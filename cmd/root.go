// Package cmd implements the generateql command-line interface.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command. Subcommands (introspect, generate) are registered
// from their own files via init().
var rootCmd = &cobra.Command{
	Use:   "generateql",
	Short: "Generate a typed Go GraphQL client library from an endpoint",
	Long: `generateql introspects a GraphQL endpoint (or a cached schema) and generates a
typed Go client library — row models, a predicate DSL for filters, native create/update
inputs, and one method per query, mutation, and subscription — running on the oh-tarnished
runtime.`,
}

// Execute runs the root command with the given build metadata, exiting non-zero on error.
func Execute(version, commit, date string) {
	rootCmd.Version = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
