package census

import (
	"go/ast"
	"go/types"
	"strings"
	"unicode"
	"unicode/utf8"
)

// recordTestFunction classifies one top-level test-scope function under
// the toolchain's test-discovery contract: the name pattern AND the exact
// signature must both match. Names never trigger semantics — a function
// merely spelled TestX with the wrong signature is ordinary code.
func recordTestFunction(info *types.Info, d *ast.FuncDecl, id, file string, line int, stats *fileStats) error {
	object, ok := info.Defs[d.Name].(*types.Func)
	if !ok {
		return nil
	}
	signature := object.Type().(*types.Signature)
	kind, err := testFunctionKind(d.Name.Name, signature)
	if err != nil {
		return err
	}
	if kind == "" {
		return nil
	}
	stats.testFunctions = append(stats.testFunctions, TestFunctionRecord{
		ID: id, Kind: kind, File: file, Line: line,
	})
	return nil
}

func testFunctionKind(name string, signature *types.Signature) (string, error) {
	if signature.Results().Len() != 0 || signature.Recv() != nil {
		return "", nil
	}
	paramType := ""
	if signature.Params().Len() == 1 {
		// Canonicalization failure PROPAGATES: a test signature whose
		// parameter type has no exact identity is a hard error, never
		// silently treated as "not a test".
		id, err := typeString(signature.Params().At(0).Type())
		if err != nil {
			return "", err
		}
		paramType = id
	} else if signature.Params().Len() != 0 {
		return "", nil
	}

	switch {
	case name == "TestMain" && paramType == "*testing.M":
		return "testmain", nil
	case matchesDiscoveryName(name, "Test") && paramType == "*testing.T":
		return "test", nil
	case matchesDiscoveryName(name, "Benchmark") && paramType == "*testing.B":
		return "benchmark", nil
	case matchesDiscoveryName(name, "Fuzz") && paramType == "*testing.F":
		return "fuzz", nil
	case matchesDiscoveryName(name, "Example") && paramType == "":
		return "example", nil
	}
	return "", nil
}

// matchesDiscoveryName follows the go test contract: the prefix must be
// followed by nothing or by a rune that is not a lower-case letter.
func matchesDiscoveryName(name, prefix string) bool {
	rest, ok := strings.CutPrefix(name, prefix)
	if !ok {
		return false
	}
	if rest == "" {
		return true
	}
	first, _ := utf8.DecodeRuneInString(rest)
	return !unicode.IsLower(first)
}
