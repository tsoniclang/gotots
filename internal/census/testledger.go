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
func recordTestFunction(info *types.Info, d *ast.FuncDecl, id, file string, line int, stats *fileStats) {
	object, ok := info.Defs[d.Name].(*types.Func)
	if !ok {
		return
	}
	signature := object.Type().(*types.Signature)
	kind := testFunctionKind(d.Name.Name, signature)
	if kind == "" {
		return
	}
	stats.testFunctions = append(stats.testFunctions, TestFunctionRecord{
		ID: id, Kind: kind, File: file, Line: line,
	})
}

func testFunctionKind(name string, signature *types.Signature) string {
	if signature.Results().Len() != 0 || signature.Recv() != nil {
		return ""
	}
	paramType := ""
	if signature.Params().Len() == 1 {
		paramType = typeString(signature.Params().At(0).Type())
	} else if signature.Params().Len() != 0 {
		return ""
	}

	switch {
	case name == "TestMain" && paramType == "*testing.M":
		return "testmain"
	case matchesDiscoveryName(name, "Test") && paramType == "*testing.T":
		return "test"
	case matchesDiscoveryName(name, "Benchmark") && paramType == "*testing.B":
		return "benchmark"
	case matchesDiscoveryName(name, "Fuzz") && paramType == "*testing.F":
		return "fuzz"
	case matchesDiscoveryName(name, "Example") && paramType == "":
		return "example"
	}
	return ""
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
