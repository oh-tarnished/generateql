package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/oh-tarnished/generate-ql/internal/introspect"
	"github.com/spf13/cobra"
)

// introspectCmd fetches a GraphQL endpoint's introspection schema and writes it to
// stdout or a file, for caching or inspection.
var introspectCmd = &cobra.Command{
	Use:   "introspect",
	Short: "Fetch a GraphQL endpoint's introspection schema",
	Long: `Introspect runs the standard GraphQL introspection query against an endpoint
and prints the resulting __schema as JSON. The output can be cached and later fed to
'generate --schema' to avoid re-querying the server.`,
	RunE: runIntrospect,
}

var (
	flagEndpoint    string
	flagHeaders     []string
	flagAdminSecret string
	flagOutput      string
)

func init() {
	rootCmd.AddCommand(introspectCmd)
	introspectCmd.Flags().StringVar(&flagEndpoint, "endpoint", "", "GraphQL endpoint URL (e.g. http://localhost:3280/graphql)")
	introspectCmd.Flags().StringArrayVar(&flagHeaders, "header", nil, "extra request header as 'Key: Value' (repeatable)")
	introspectCmd.Flags().StringVar(&flagAdminSecret, "admin-secret", "", "shortcut for the x-hasura-admin-secret header")
	introspectCmd.Flags().StringVarP(&flagOutput, "out", "o", "", "write schema JSON to this file (default stdout)")
	_ = introspectCmd.MarkFlagRequired("endpoint")
}

func runIntrospect(cmd *cobra.Command, _ []string) error {
	schema, err := introspect.Fetch(context.Background(), flagEndpoint, buildHeaders())
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode schema: %w", err)
	}
	if flagOutput == "" {
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return nil
	}
	if err := os.WriteFile(flagOutput, out, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", flagOutput, err)
	}
	fmt.Fprintf(cmd.OutOrStderr(), "wrote schema to %s\n", flagOutput)
	return nil
}

// buildHeaders merges --admin-secret and repeated --header flags into a header map.
func buildHeaders() map[string]string {
	headers := map[string]string{}
	if flagAdminSecret != "" {
		headers["x-hasura-admin-secret"] = flagAdminSecret
	}
	for _, h := range flagHeaders {
		key, value, ok := strings.Cut(h, ":")
		if !ok {
			continue
		}
		headers[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return headers
}
