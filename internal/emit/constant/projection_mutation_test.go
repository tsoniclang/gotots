package constant_test

import (
	"errors"
	"go/ast"
	"go/types"
	"slices"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

// TestUntypedConstantProjectionDedupsByRepresentation proves projections are
// keyed by (constant, representation): three functions use Scale at overlapping
// representations — MultipleTargets at int8/int32/int64/uint16, Argument at
// int32, Assignment at uint16 — yet each distinct representation is declared
// exactly once. A per-use rather than per-representation projection would emit
// duplicates.
func TestUntypedConstantProjectionDedupsByRepresentation(t *testing.T) {
	loaded := loadConstantFamily(t)
	emission := compileConstantFamily(
		t, loaded, emit.DefaultOptions(),
		"MultipleTargets", "Argument", "Assignment",
	)
	source := constantFamilySourceFile(t, emission)

	var scaleProjections []string
	for _, name := range declaredTopLevelBindings(source) {
		if strings.HasPrefix(name, "Scale$") {
			scaleProjections = append(scaleProjections, name)
		}
	}
	slices.Sort(scaleProjections)
	want := []string{"Scale$int32", "Scale$int64", "Scale$int8", "Scale$uint16"}
	if !slices.Equal(scaleProjections, want) {
		t.Fatalf("Scale projections = %v, want each distinct representation exactly once %v",
			scaleProjections, want)
	}
}

// TestConstantUseRequiresSemanticEvidence proves the projection path selects the
// constant from the checker's Uses evidence, not from the identifier spelling:
// removing only the semantic Uses entry of a Scale use — its spelling and every
// other fact untouched — fails at the identifier handler's first gate rather
// than recovering the constant by name.
func TestConstantUseRequiresSemanticEvidence(t *testing.T) {
	loaded := loadConstantFamily(t)
	useIdent := findConstUse(t, loaded, "Argument", "Scale")
	delete(loaded.TypesInfo().Uses, useIdent)

	object := loaded.Types().Scope().Lookup("Argument")
	root, err := emit.NewRoot(object)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(loaded.Program(), []emit.Root{root})
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != "*ast.Ident" {
		t.Fatalf("error = %#v, want expression UnsupportedError at the identifier gate", err)
	}
}

// findConstUse returns the identifier of a use of the named constant inside the
// named function body.
func findConstUse(t *testing.T, loaded *load.Package, function, constant string) *ast.Ident {
	t.Helper()
	var found *ast.Ident
	for _, file := range loaded.Files() {
		for _, declaration := range file.Syntax().Decls {
			decl, ok := declaration.(*ast.FuncDecl)
			if !ok || decl.Name.Name != function {
				continue
			}
			ast.Inspect(decl.Body, func(node ast.Node) bool {
				identifier, ok := node.(*ast.Ident)
				if !ok || identifier.Name != constant {
					return true
				}
				if _, isConst := loaded.TypesInfo().Uses[identifier].(*types.Const); isConst {
					found = identifier
					return false
				}
				return true
			})
		}
	}
	if found == nil {
		t.Fatalf("no use of constant %s in function %s", constant, function)
	}
	return found
}
