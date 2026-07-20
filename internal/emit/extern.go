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

// Stub symbol spellings: "$"-suffixed member names no Go identifier can
// collide with. Generated call sites and stub exports must agree.
func externMethodSymbol(typeName, method string) string { return typeName + "$" + method }
func externZeroSymbol(typeName string) string           { return typeName + "$goZero$" }
func externCloneSymbol(typeName string) string          { return typeName + "$goClone$" }
func externSetSymbol(typeName string) string            { return typeName + "$goSet$" }
func externVarSymbol(name string) string                { return name + "$get$" }

// ExternMethodSymbol exposes the method-stub spelling for call sites
// outside the emit package.
func ExternMethodSymbol(typeName, method string) string { return externMethodSymbol(typeName, method) }

// StubModule prints one external package's typed stubs: package-level
// functions, then type members and variables, all failing closed.
func StubModule(module *Module, funcs []StubFunc, members []StubMember) (string, error) {
	var body strings.Builder
	for _, fn := range funcs {
		body.WriteString("\n")
		if err := printStubFunc(&body, module, fn); err != nil {
			return "", err
		}
	}
	for _, member := range members {
		body.WriteString("\n")
		if err := printStubMember(&body, module, member); err != nil {
			return "", err
		}
	}
	// Build every reserved union alias's deferred definition after all
	// external adapter types exist (see finalizeUnionAliases).
	if err := finalizeUnionAliases(module); err != nil {
		return "", err
	}
	return module.importLines() + module.aliasLines() + module.eqFnLines() + body.String(), nil
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
		// A generic external contract carries the factory quadruple per
		// type parameter — the same per-binding operations owned generic
		// signatures take — so implementations reproduce Go's value-copy
		// semantics for every binding.
		for _, param := range fn.TypeParams {
			params = append(params, "zero$"+param+": () => "+param)
		}
		for _, param := range fn.TypeParams {
			params = append(params, "eq$"+param+": (a: "+param+", b: "+param+") => boolean")
		}
		for _, param := range fn.TypeParams {
			params = append(params, "clone$"+param+": (v: "+param+") => "+param)
		}
		for _, param := range fn.TypeParams {
			params = append(params, "set$"+param+": ((d: "+param+", s: "+param+") => void) | undefined")
		}
	}

	_ = names
	module.recordExternSymbol(fn.Name, fn.ID)
	p.line("export function %s%s(%s): %s {", fn.Name, generics, strings.Join(params, ", "), result)
	p.indent++
	// The stub is the typed contract with no behavior: it fails closed
	// until assembly replaces this module with an implementation module
	// exporting the same statically selected symbols.
	p.line("return goext$.goExternalUnimplemented(%q);", fn.ID)
	p.indent--
	p.line("}")
	return nil
}

// StubMember is one external type-member or variable contract: a typed
// export whose stub body fails closed.
type StubMember struct {
	ID     string // canonical diagnostic identity
	Name   string // exported stub symbol
	Params []ir.Var
	// ResultType, when set, is a single result; Results is the general
	// list. Both empty means void.
	ResultType *ir.Type
	Results    []ir.Type
}

// printStubMember prints one member stub.
func printStubMember(out *strings.Builder, module *Module, member StubMember) error {
	p := &printer{out: out, module: module}
	params := make([]string, 0, len(member.Params))
	for i, parameter := range member.Params {
		name := tsName(parameter.Name)
		if parameter.Name == "" || parameter.Name == "_" {
			name = fmt.Sprintf("p%d", i)
		}
		spelled, err := p.tsType(parameter.Type)
		if err != nil {
			return fmt.Errorf("%s: %w", member.ID, err)
		}
		params = append(params, name+": "+spelled)
	}
	results := member.Results
	if member.ResultType != nil {
		results = []ir.Type{*member.ResultType}
	}
	result, err := p.tsFuncResultType(results)
	if err != nil {
		return fmt.Errorf("%s: %w", member.ID, err)
	}
	module.recordExternSymbol(member.Name, member.ID)
	p.line("export function %s(%s): %s {", member.Name, strings.Join(params, ", "), result)
	p.indent++
	p.line("return goext$.goExternalUnimplemented(%q);", member.ID)
	p.indent--
	p.line("}")
	return nil
}
