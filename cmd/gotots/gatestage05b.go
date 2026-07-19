// Acceptance stage 05, independence half: the emitted TypeScript
// declarations are PARSED with the pinned compiler and compared
// structurally with the census's Go declaration shapes — never through
// the shared canonical renderer. The prediction encodes only the
// REVIEWED lowering contract (receiver-first parameter, the
// zero/eq/clone/set/key factory quintuple per type parameter, blank
// zero-size fields as no-output, the eq/clone/set capture triple on
// generic classes).
package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/declparity"
	"github.com/tsoniclang/gotots/internal/translate"
)

// runDeclParityCheck compares every module-retained declaration's parsed
// TypeScript structure with its census Go shape. It returns the number
// of independently verified declarations plus any structural defects.
func runDeclParityCheck(firstRun *census.Result, corpusGenerated *translate.Generated, repoDir string) (int, []string, error) {
	typescriptModule := filepath.Join(repoDir, "product", "node_modules", "typescript")
	moduleFiles := map[string]string{}
	for path, content := range corpusGenerated.Files {
		if corpusGenerated.Ownership[path] == "generated-core" {
			moduleFiles[path] = content
		}
	}
	parsed, err := declparity.Extract(moduleFiles, typescriptModule)
	if err != nil {
		return 0, nil, err
	}
	proofs := map[string]*translate.Proof{}
	for i := range corpusGenerated.Proofs {
		proofs[corpusGenerated.Proofs[i].ID] = &corpusGenerated.Proofs[i]
	}
	production := map[string]bool{}
	for _, decl := range firstRun.Report.Declarations {
		if decl.Scope == "production" {
			production[decl.ID] = true
		}
	}
	var defects []string
	defect := func(format string, args ...any) {
		if len(defects) < 20 {
			defects = append(defects, fmt.Sprintf(format, args...))
		}
	}
	verified := 0

	// Functions and methods: the declared parameter list is receiver
	// (when present) + source parameters + the factory quintuple per type
	// parameter (zero/eq/clone/set/key).
	for _, shape := range firstRun.Shapes.Functions {
		if !production[shape.ID] {
			continue
		}
		proof, has := proofs[shape.ID]
		if !has || !proof.ModuleRetained || proof.GeneratedSymbol == "" {
			continue
		}
		decl, found := parsed[proof.GeneratedFile][proof.GeneratedSymbol]
		if !found {
			defect("declared symbol %s absent from parsed %s (census %s)", proof.GeneratedSymbol, proof.GeneratedFile, shape.ID)
			continue
		}
		if decl.Kind != "function" {
			defect("census function %s parsed as %s", shape.ID, decl.Kind)
			continue
		}
		valueParams := len(shape.Params)
		if shape.Receiver != "" {
			valueParams++
		}
		// The reviewed factory protocol, verified STRUCTURALLY from the
		// parsed parameter names: after the value parameters, one
		// zero$/eq$/clone$/set$ slot per DECLARED (surface) type
		// parameter in declaration order, then the requirement-scoped
		// key$ and rt$ groups over subsets of them. Surface parameters
		// may be FEWER than the Go declaration's (core-typed parameters
		// erase), never more.
		if decl.TypeParams > len(shape.TypeParams) {
			defect("type-parameter arity drift at %s: parsed %d, Go declares %d", shape.ID, decl.TypeParams, len(shape.TypeParams))
			continue
		}
		if problem := factoryProtocolDefect(decl, valueParams); problem != "" {
			defect("factory-protocol drift at %s: %s", shape.ID, problem)
			continue
		}
		verified++
	}

	// Named types: structs parse as classes carrying every non-blank Go
	// field as a property (plus the capture triple per type parameter);
	// non-struct named types parse as type aliases with matching type
	// parameters.
	for _, shape := range firstRun.Shapes.Types {
		if !production[shape.ID] {
			continue
		}
		proof, has := proofs[shape.ID]
		if !has || !proof.ModuleRetained || proof.GeneratedSymbol == "" {
			continue
		}
		decl, found := parsed[proof.GeneratedFile][proof.GeneratedSymbol]
		if !found {
			defect("declared type %s absent from parsed %s (census %s)", proof.GeneratedSymbol, proof.GeneratedFile, shape.ID)
			continue
		}
		if shape.Kind == "struct" {
			if decl.Kind != "class" {
				defect("census struct %s parsed as %s", shape.ID, decl.Kind)
				continue
			}
			if decl.TypeParams != len(shape.TypeParams) {
				defect("class type-parameter arity drift at %s: parsed %d, Go declares %d", shape.ID, decl.TypeParams, len(shape.TypeParams))
				continue
			}
			properties := map[string]bool{}
			for _, field := range decl.Fields {
				properties[field] = true
			}
			complete := true
			for _, field := range shape.Fields {
				if field.Name == "_" {
					// Blank zero-size fields are no-output by the reviewed
					// lowering.
					continue
				}
				if !properties[field.Name] {
					defect("struct field %s.%s absent from parsed class properties", shape.ID, field.Name)
					complete = false
				}
			}
			for _, param := range shape.TypeParams {
				for _, capture := range []string{"eq$", "clone$", "set$"} {
					if !properties[capture+param.Name] {
						defect("generic capture %s%s absent from parsed class %s", capture, param.Name, shape.ID)
						complete = false
					}
				}
			}
			if !complete {
				continue
			}
			verified++
			continue
		}
		if decl.Kind != "type" {
			defect("census named type %s parsed as %s", shape.ID, decl.Kind)
			continue
		}
		if decl.TypeParams != len(shape.TypeParams) {
			defect("carrier type-parameter arity drift at %s: parsed %d, Go declares %d", shape.ID, decl.TypeParams, len(shape.TypeParams))
			continue
		}
		verified++
	}

	// Variables: a retained non-effect-only variable declares a module
	// binding.
	for _, shape := range firstRun.Shapes.Variables {
		if !production[shape.ID] {
			continue
		}
		proof, has := proofs[shape.ID]
		if !has || !proof.ModuleRetained || proof.GeneratedSymbol == "" || proof.EffectOnly {
			continue
		}
		decl, found := parsed[proof.GeneratedFile][proof.GeneratedSymbol]
		if !found {
			defect("declared variable %s absent from parsed %s (census %s)", proof.GeneratedSymbol, proof.GeneratedFile, shape.ID)
			continue
		}
		if decl.Kind != "let" && decl.Kind != "const" {
			defect("census variable %s parsed as %s", shape.ID, decl.Kind)
			continue
		}
		verified++
	}
	return verified, defects, nil
}

// factoryProtocolDefect verifies a parsed generic function's trailing
// parameters against the reviewed factory protocol using ONLY the
// parsed structure: value parameters first, then zero$P eq$P clone$P
// set$P for every surface type parameter (each group in type-parameter
// order), then the requirement-scoped key$P and rt$P groups over
// subsets in the same order. Returns "" when conformant.
func factoryProtocolDefect(decl declparity.TSDecl, valueParams int) string {
	if len(decl.ParamNames) != decl.ParamCount {
		return fmt.Sprintf("parsed %d parameters but %d names", decl.ParamCount, len(decl.ParamNames))
	}
	if len(decl.ParamNames) < valueParams {
		return fmt.Sprintf("parsed %d parameters, Go shape implies at least %d value parameters", len(decl.ParamNames), valueParams)
	}
	trailing := decl.ParamNames[valueParams:]
	surface := decl.TypeParamNames
	index := 0
	for _, prefix := range []string{"zero$", "eq$", "clone$", "set$"} {
		for _, param := range surface {
			if index >= len(trailing) || trailing[index] != prefix+param {
				got := "(absent)"
				if index < len(trailing) {
					got = trailing[index]
				}
				return fmt.Sprintf("expected %s%s at trailing slot %d, parsed %s", prefix, param, index, got)
			}
			index++
		}
	}
	for _, prefix := range []string{"key$", "rt$"} {
		// A subset of the surface parameters, in declaration order.
		cursor := 0
		for index < len(trailing) && strings.HasPrefix(trailing[index], prefix) {
			name := strings.TrimPrefix(trailing[index], prefix)
			for cursor < len(surface) && surface[cursor] != name {
				cursor++
			}
			if cursor >= len(surface) {
				return fmt.Sprintf("%s slot %q is not a declared type parameter in order", prefix, trailing[index])
			}
			cursor++
			index++
		}
	}
	if index != len(trailing) {
		return fmt.Sprintf("unexpected trailing parameter %q at slot %d", trailing[index], index)
	}
	return ""
}
