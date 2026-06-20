package golang

import (
	"path"
	"strings"
)

// Type packages: each generated GraphQL type gets its own file, grouped by kind into
// separate packages so references stay cycle-free.
const (
	modelsDir = "types/schema" // object/row models package, referenced as "schema."
	inputsDir = "types/inputs"
	enumsDir  = "types/enums"
)

// writeTypes writes one file per GraphQL type, named after the type, into the enums,
// models (schema), and inputs packages.
func (g *generator) writeTypes() error {
	for _, name := range sortedKeys(g.opts.Schema.Enums) {
		body := g.r.enum(g.opts.Schema.Enums[name])
		if err := g.writeFile(enumsDir, typeFile(name), "enums", g.typeImports(body), body); err != nil {
			return err
		}
	}
	for _, name := range sortedKeys(g.opts.Schema.Objects) {
		body := g.r.model(g.opts.Schema.Objects[name])
		if err := g.writeFile(modelsDir, typeFile(name), "schema", g.typeImports(body), body); err != nil {
			return err
		}
	}
	for _, name := range sortedKeys(g.opts.Schema.Inputs) {
		body := g.r.input(g.opts.Schema.Inputs[name])
		if err := g.writeFile(inputsDir, typeFile(name), "inputs", g.typeImports(body), body); err != nil {
			return err
		}
	}
	return nil
}

// typeFile returns the file name for a generated type: the schema type name itself,
// kept PascalCase so it contains no underscores. Underscores would let Go misread a
// trailing word as a GOOS/GOARCH build constraint (e.g. "..._windows.go").
func typeFile(name string) string {
	return name + ".go"
}

// typeImports returns the imports a file needs based on the qualified type references
// and json/graphql usage in its body.
func (g *generator) typeImports(body string) []string {
	var im []string
	if strings.Contains(body, "json.RawMessage") {
		im = append(im, "encoding/json")
	}
	if strings.Contains(body, "graphql.") {
		im = append(im, g.graphqlModule())
	}
	if strings.Contains(body, "param.") {
		im = append(im, g.paramModule())
	}
	if strings.Contains(body, "enums.") {
		im = append(im, g.opts.GoModule+"/"+enumsDir)
	}
	if strings.Contains(body, "inputs.") {
		im = append(im, g.opts.GoModule+"/"+inputsDir)
	}
	if strings.Contains(body, "schema.") {
		im = append(im, g.opts.GoModule+"/"+modelsDir)
	}
	return im
}

// graphqlModule is the import path of the runtime graphql helper package (scalar types
// and pointer constructors), a sibling of the runtime facade module.
func (g *generator) graphqlModule() string {
	return path.Dir(g.opts.RuntimeModule) + "/graphql"
}

// paramModule is the import path of the runtime param package (Opt and IsOmitted), a
// sibling of the runtime facade module.
func (g *generator) paramModule() string {
	return path.Dir(g.opts.RuntimeModule) + "/param"
}
