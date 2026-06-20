package golang

import (
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/oh-tarnished/generateql/internal/ir"
	"github.com/oh-tarnished/generateql/internal/selection"
	"github.com/oh-tarnished/generateql/internal/typemap"
)

//go:embed templates/file.go.tmpl
var templatesFS embed.FS

// Options configures Go client generation.
type Options struct {
	Schema        *ir.Schema
	OutDir        string
	Package       string            // root package name (Service + New)
	GoModule      string            // import path of the generated root package
	RuntimeModule string            // import path of the runtime facade
	MaxDepth      int               // relation inlining depth
	Scalars       map[string]string // GraphQL scalar -> Go type overrides
}

// fileData is the data passed to the file template.
type fileData struct {
	Package string
	Imports []string
	Body    string
}

// generator holds shared state across the write_*.go output passes.
type generator struct {
	opts    Options
	tmpl    *template.Template
	r       *renderer
	domains []*domainGen
}

// Generate renders the full Go client into Options.OutDir: the shared type packages,
// per-resource handler packages, per-domain aggregators, and the root Service.
func Generate(opts Options) error {
	tmpl, err := template.ParseFS(templatesFS, "templates/file.go.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	mapper := typemap.New(opts.Schema, opts.Scalars)
	g := &generator{
		opts: opts,
		tmpl: tmpl,
		r: &renderer{
			schema:    opts.Schema,
			mapper:    mapper,
			selection: selection.New(opts.Schema, mapper, opts.MaxDepth, qModels),
		},
	}
	g.plan()

	if err := g.writeTypes(); err != nil {
		return err
	}
	if err := g.writeResources(); err != nil {
		return err
	}
	if err := g.writeDomains(); err != nil {
		return err
	}
	if err := g.writeHelpers(); err != nil {
		return err
	}
	return g.writeRoot()
}

// writeFile renders one Go file through the template, gofmt-formats it, and writes it
// to <OutDir>/<subdir>/<name>.
func (g *generator) writeFile(subdir, name, pkg string, imports []string, body string) error {
	dir := filepath.Join(g.opts.OutDir, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}
	var raw strings.Builder
	if err := g.tmpl.Execute(&raw, fileData{Package: pkg, Imports: imports, Body: body}); err != nil {
		return fmt.Errorf("template exec for %s: %w", name, err)
	}
	formatted, err := format.Source([]byte(raw.String()))
	if err != nil {
		return fmt.Errorf("gofmt %s/%s: %w", subdir, name, err)
	}
	return os.WriteFile(filepath.Join(dir, name), formatted, 0o644)
}
