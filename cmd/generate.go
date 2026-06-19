package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/oh-tarnished/generate-ql/internal/gen/golang"
	"github.com/oh-tarnished/generate-ql/internal/introspect"
	"github.com/oh-tarnished/generate-ql/internal/ir"
	"github.com/spf13/cobra"
)

// defaultRuntimeModule is the import path of the Go runtime facade that generated
// clients depend on.
const defaultRuntimeModule = "github.com/oh-tarnished/generate-ql/runtime/go/runtime"

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a typed GraphQL client from an endpoint or cached schema",
	Long: `Generate introspects a GraphQL endpoint (or reads a cached --schema file),
then writes a typed Go client — models, inputs, enums, and one function per query and
mutation — into the output directory.`,
	RunE: runGenerate,
}

var (
	flagGenEndpoint string
	flagGenHeaders  []string
	flagGenSecret   string
	flagSchemaFile  string
	flagOutDir      string
	flagPackage     string
	flagGoModule    string
	flagRuntimeMod  string
	flagMaxDepth    int
	flagScalars     []string
)

func init() {
	rootCmd.AddCommand(generateCmd)
	f := generateCmd.Flags()
	f.StringVar(&flagGenEndpoint, "endpoint", "", "GraphQL endpoint URL (used when --schema is not set)")
	f.StringVar(&flagSchemaFile, "schema", "", "path to a cached introspection JSON file")
	f.StringArrayVar(&flagGenHeaders, "header", nil, "extra request header as 'Key: Value' (repeatable)")
	f.StringVar(&flagGenSecret, "admin-secret", "", "shortcut for the x-hasura-admin-secret header")
	f.StringVarP(&flagOutDir, "out", "o", "./generated", "output directory for the generated client")
	f.StringVar(&flagPackage, "package", "client", "Go package name for the generated root package")
	f.StringVar(&flagGoModule, "go-module", "", "import path of the generated root package (required; resource packages import <go-module>/types)")
	f.StringVar(&flagRuntimeMod, "runtime-module", defaultRuntimeModule, "import path of the runtime facade")
	f.IntVar(&flagMaxDepth, "max-depth", 1, "how many levels of relations to inline into models")
	f.StringArrayVar(&flagScalars, "scalar", nil, "scalar override as 'GraphQLName=GoType' (repeatable)")
}

func runGenerate(cmd *cobra.Command, _ []string) error {
	schema, err := loadSchema()
	if err != nil {
		return err
	}

	if flagGoModule == "" {
		return fmt.Errorf("--go-module is required (the import path of the generated package)")
	}

	model := ir.Build(schema)
	opts := golang.Options{
		Schema:        model,
		OutDir:        flagOutDir,
		Package:       flagPackage,
		GoModule:      flagGoModule,
		RuntimeModule: flagRuntimeMod,
		MaxDepth:      flagMaxDepth,
		Scalars:       parseScalars(flagScalars),
	}
	if err := golang.Generate(opts); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStderr(), "generated %d objects, %d queries, %d mutations, %d subscriptions into %s\n",
		len(model.Objects), len(model.Queries), len(model.Mutations), len(model.Subscriptions), flagOutDir)
	return nil
}

// loadSchema reads the introspection schema from --schema or fetches it live.
func loadSchema() (*introspect.Schema, error) {
	if flagSchemaFile != "" {
		raw, err := os.ReadFile(flagSchemaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read schema file: %w", err)
		}
		return introspect.Decode(raw)
	}
	if flagGenEndpoint == "" {
		return nil, fmt.Errorf("either --endpoint or --schema is required")
	}
	headers := map[string]string{}
	if flagGenSecret != "" {
		headers["x-hasura-admin-secret"] = flagGenSecret
	}
	for _, h := range flagGenHeaders {
		if key, value, ok := strings.Cut(h, ":"); ok {
			headers[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return introspect.Fetch(context.Background(), flagGenEndpoint, headers)
}

// parseScalars converts 'Name=GoType' entries into a map.
func parseScalars(entries []string) map[string]string {
	out := map[string]string{}
	for _, e := range entries {
		if key, value, ok := strings.Cut(e, "="); ok {
			out[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return out
}
