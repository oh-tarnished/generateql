package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/oh-tarnished/generateql/internal/gen/golang"
	"github.com/oh-tarnished/generateql/internal/introspect"
	"github.com/oh-tarnished/generateql/internal/ir"
	"github.com/spf13/cobra"
)

// defaultRuntimeModule is the import path of the Go runtime facade that generated
// clients depend on. It lives in the shared runtime-go repo (network module);
// the sibling predicate DSL package is derived from it as path.Dir + "/graphql".
const defaultRuntimeModule = "github.com/the-protobuf-project/runtime-go/network/runtime"

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a typed Go GraphQL client library from an endpoint or cached schema",
	Long: `Generate introspects a GraphQL endpoint (or reads a cached --schema file), then
writes a typed Go client library — row models, a predicate DSL, native create/update inputs,
and one method per query, mutation, and subscription — into <out>/<package>/.`,
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
	flagLang        string
	flagConfig      string
	flagDumpSchema  bool
)

func init() {
	rootCmd.AddCommand(generateCmd)
	f := generateCmd.Flags()
	f.StringVar(&flagConfig, "config", "", "path to a generateql.yaml config (auto-detected in the working directory if omitted)")
	f.StringVar(&flagLang, "lang", "go", "target language (currently only 'go')")
	f.StringVar(&flagGenEndpoint, "endpoint", "", "GraphQL endpoint URL to introspect live (used when --schema is not set)")
	f.StringVar(&flagSchemaFile, "schema", "", "path to a cached introspection JSON file (skips the live introspection)")
	f.StringArrayVar(&flagGenHeaders, "header", nil, "extra request header as 'Key: Value' (repeatable)")
	f.StringVar(&flagGenSecret, "admin-secret", "", "shortcut for the x-hasura-admin-secret header")
	f.StringVarP(&flagOutDir, "out", "o", ".", "parent directory; the client is written into <out>/<package>/ (a library inside your existing module, protobuf-style)")
	f.StringVar(&flagPackage, "package", "", "Go package name for the root package (default: last segment of --go-module + \"ql\", e.g. freebusy -> freebusyql)")
	f.StringVar(&flagGoModule, "go-module", "", "import path of the generated root package (required; subpackages live under <go-module>/<domain>ql/...)")
	f.StringVar(&flagRuntimeMod, "runtime-module", defaultRuntimeModule, "import path of the runtime facade")
	f.IntVar(&flagMaxDepth, "max-depth", 1, "how many levels of relations to inline into models")
	f.StringArrayVar(&flagScalars, "scalar", nil, "scalar override as 'GraphQLName=GoType' (repeatable)")
	f.BoolVar(&flagDumpSchema, "dump-schema", false, "also write the introspection schema as <package>/schema.json")
}

func runGenerate(cmd *cobra.Command, _ []string) error {
	if err := applyConfig(cmd); err != nil {
		return err
	}
	if flagLang != "go" {
		return fmt.Errorf("unsupported --lang %q (only 'go' is currently supported)", flagLang)
	}

	schema, err := loadSchema()
	if err != nil {
		return err
	}

	if flagGoModule == "" {
		return fmt.Errorf("--go-module is required (the import path of the generated package)")
	}

	pkg := flagPackage
	if pkg == "" {
		pkg = path.Base(flagGoModule)
		if !strings.HasSuffix(pkg, "ql") {
			pkg += "ql"
		}
	}

	model := ir.Build(schema)
	opts := golang.Options{
		Schema:        model,
		OutDir:        flagOutDir,
		Package:       pkg,
		GoModule:      flagGoModule,
		RuntimeModule: flagRuntimeMod,
		MaxDepth:      flagMaxDepth,
		Scalars:       parseScalars(flagScalars),
	}
	if err := golang.Generate(opts); err != nil {
		return err
	}
	if flagDumpSchema {
		if err := dumpSchema(schema, filepath.Join(flagOutDir, pkg)); err != nil {
			return err
		}
	}
	_, _ = fmt.Fprintf(cmd.OutOrStderr(), "generated %d objects, %d queries, %d mutations, %d subscriptions into %s\n",
		len(model.Objects), len(model.Queries), len(model.Mutations), len(model.Subscriptions), filepath.Join(flagOutDir, pkg))
	return nil
}

// dumpSchema writes the introspection schema the client was generated from into the
// generated package directory, so the library carries its own source of truth.
func dumpSchema(schema *introspect.Schema, dir string) error {
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode schema dump: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "schema.json"), data, 0o644)
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
