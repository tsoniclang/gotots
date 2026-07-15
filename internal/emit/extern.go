package emit

import (
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
)

// StubFunc is one external function contract: a fully typed signature
// whose body delegates to the external-contract registry by canonical
// identity, failing closed at runtime unless the emulation layer
// registered behavior.
type StubFunc struct {
	ID         string // canonical "<package path>.<Name>" registry key
	Name       string
	TypeParams []string
	Params     []ir.Var
	Results    []ir.Var
}

// StubModule prints one external package's typed stubs.
func StubModule(module *Module, funcs []StubFunc) (string, error) {
	var body strings.Builder
	for _, fn := range funcs {
		body.WriteString("\n")
		if err := printStubFunc(&body, module, fn); err != nil {
			return "", err
		}
	}
	return module.importLines() + body.String(), nil
}

func printStubFunc(out *strings.Builder, module *Module, fn StubFunc) error {
	p := &printer{out: out, module: module}
	params := make([]string, 0, len(fn.Params))
	names := make([]string, 0, len(fn.Params))
	for i, parameter := range fn.Params {
		name := tsName(parameter.Name)
		if parameter.Name == "" || parameter.Name == "_" {
			name = fmt.Sprintf("p%d", i)
		}
		spelled, err := p.tsType(parameter.Type)
		if err != nil {
			return fmt.Errorf("%s: %w", fn.ID, err)
		}
		params = append(params, name+": "+spelled)
		names = append(names, name)
	}
	result, err := p.tsResultType(fn.Results)
	if err != nil {
		return fmt.Errorf("%s: %w", fn.ID, err)
	}
	generics := ""
	if len(fn.TypeParams) > 0 {
		generics = "<" + strings.Join(fn.TypeParams, ", ") + ">"
	}

	resultTypes := make([]ir.Type, len(fn.Results))
	for i, r := range fn.Results {
		resultTypes[i] = r.Type
	}
	call := fmt.Sprintf("goext$.goExternalCall(%q, [%s])", fn.ID, strings.Join(names, ", "))
	cast, err := p.castResults(call, resultTypes)
	if err != nil {
		return fmt.Errorf("%s: %w", fn.ID, err)
	}

	p.line("export function %s%s(%s): %s {", fn.Name, generics, strings.Join(params, ", "), result)
	p.indent++
	if len(fn.Results) == 0 {
		p.line("%s;", cast)
	} else {
		p.line("return %s;", cast)
	}
	p.indent--
	p.line("}")
	return nil
}
