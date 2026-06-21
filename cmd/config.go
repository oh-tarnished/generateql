package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// genConfig mirrors the generate flags so a project can keep its settings in a
// generateql.yaml file and run `generateql generate` with no arguments. Any flag passed
// explicitly on the command line overrides the corresponding config value.
type genConfig struct {
	Endpoint    string   `yaml:"endpoint"`
	Schema      string   `yaml:"schema"`
	Lang        string   `yaml:"lang"`
	GoModule    string   `yaml:"go-module"`
	Out         string   `yaml:"out"`
	Package     string   `yaml:"package"`
	RuntimeMod  string   `yaml:"runtime-module"`
	MaxDepth    *int     `yaml:"max-depth"`
	Headers     []string `yaml:"headers"`
	AdminSecret string   `yaml:"admin-secret"`
	Scalars     []string `yaml:"scalars"`
	DumpSchema  *bool    `yaml:"dump-schema"`
}

// configNames are the config file names auto-detected in the working directory.
var configNames = []string{"generateql.yaml", "generateql.yml"}

// applyConfig loads a config file (from --config, or auto-detected) and fills every flag
// the user did not set explicitly. Explicit flags win; relative paths are resolved against
// the working directory.
func applyConfig(cmd *cobra.Command) error {
	path := flagConfig
	if path == "" {
		for _, name := range configNames {
			if _, err := os.Stat(name); err == nil {
				path = name
				break
			}
		}
	}
	if path == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg genConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}

	f := cmd.Flags()
	str := func(name, val string, dst *string) {
		if val != "" && !f.Changed(name) {
			*dst = val
		}
	}
	str("endpoint", cfg.Endpoint, &flagGenEndpoint)
	str("schema", cfg.Schema, &flagSchemaFile)
	str("lang", cfg.Lang, &flagLang)
	str("go-module", cfg.GoModule, &flagGoModule)
	str("out", cfg.Out, &flagOutDir)
	str("package", cfg.Package, &flagPackage)
	str("runtime-module", cfg.RuntimeMod, &flagRuntimeMod)
	str("admin-secret", cfg.AdminSecret, &flagGenSecret)
	if cfg.MaxDepth != nil && !f.Changed("max-depth") {
		flagMaxDepth = *cfg.MaxDepth
	}
	if cfg.DumpSchema != nil && !f.Changed("dump-schema") {
		flagDumpSchema = *cfg.DumpSchema
	}
	if len(cfg.Headers) > 0 && !f.Changed("header") {
		flagGenHeaders = cfg.Headers
	}
	if len(cfg.Scalars) > 0 && !f.Changed("scalar") {
		flagScalars = cfg.Scalars
	}
	return nil
}
