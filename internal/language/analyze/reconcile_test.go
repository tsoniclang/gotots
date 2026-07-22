package analyze

import (
	"fmt"
	"go/constant"
	"go/importer"
	"go/token"
	"go/types"
	"reflect"
	"testing"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// nodeInterfaceType is the reflect view of ast.Node.
var nodeInterfaceType = reflect.TypeOf((*interface {
	Pos() token.Pos
	End() token.Pos
})(nil)).Elem()

// isNodeType reports whether a field type holds a syntax node: a concrete
// node pointer or a node interface.
func isNodeType(t reflect.Type) bool {
	return t.Implements(nodeInterfaceType)
}

// TestEdgeCatalogReconcilesToolchainFields proves the edge catalog is total
// against the selected toolchain, not against any fixture: every exported
// node-bearing field of every active concrete node type is exactly one
// cataloged edge or one recorded exclusion, every cataloged edge exists with
// the matching shape, and the per-kind edge order equals the toolchain's
// struct field order (the visit-order contract). Deleting any edge — including
// Field.Comment — fails here.
func TestEdgeCatalogReconcilesToolchainFields(t *testing.T) {
	excluded := map[string]string{}
	for _, exclusion := range catalog.ExcludedFields() {
		excluded[exclusion.Kind.Name()+"."+exclusion.Field] = exclusion.Reason
	}
	edgeByName := map[string]catalog.Edge{}
	for _, edge := range catalog.AllEdges() {
		edgeByName[edge.Name()] = edge
	}
	coveredEdges := map[catalog.Edge]bool{}
	for typeName, instance := range astInstances {
		kind, err := Classify(instance)
		if err != nil {
			t.Fatalf("Classify(%s): %v", typeName, err)
		}
		structType := reflect.TypeOf(instance).Elem()
		if kind.Disposition() != catalog.DispositionActive {
			if len(catalog.EdgesOf(kind)) != 0 {
				t.Errorf("non-active kind %s has cataloged edges", kind)
			}
			continue
		}
		var fieldOrder []catalog.Edge
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			fieldType := field.Type
			isList := fieldType.Kind() == reflect.Slice && isNodeType(fieldType.Elem())
			isSingle := isNodeType(fieldType)
			if !isList && !isSingle {
				if fieldType.Kind() == reflect.Map &&
					(isNodeType(fieldType.Elem()) || isNodeType(fieldType.Key())) {
					t.Errorf("%s.%s: node-bearing map field on active kind", typeName, field.Name)
				}
				continue
			}
			qualified := typeName + "." + field.Name
			if _, isExcluded := excluded[qualified]; isExcluded {
				if _, alsoEdge := edgeByName[qualified]; alsoEdge {
					t.Errorf("%s is both an edge and an exclusion", qualified)
				}
				continue
			}
			edge, cataloged := edgeByName[qualified]
			if !cataloged {
				t.Errorf("toolchain node-bearing field %s is neither a cataloged edge nor a recorded exclusion", qualified)
				continue
			}
			if edge.IsList() != isList {
				t.Errorf("edge %s listness %v, toolchain field has %v", qualified, edge.IsList(), isList)
			}
			coveredEdges[edge] = true
			fieldOrder = append(fieldOrder, edge)
		}
		cataloged := catalog.EdgesOf(kind)
		if len(cataloged) != len(fieldOrder) {
			t.Errorf("kind %s: catalog has %d edges, toolchain has %d node-bearing fields", kind, len(cataloged), len(fieldOrder))
			continue
		}
		for i := range cataloged {
			if cataloged[i] != fieldOrder[i] {
				t.Errorf("kind %s edge order diverges at %d: catalog %s, toolchain %s",
					kind, i, cataloged[i], fieldOrder[i])
			}
		}
	}
	for _, edge := range catalog.AllEdges() {
		if !coveredEdges[edge] {
			t.Errorf("cataloged edge %s has no toolchain field", edge)
		}
	}
}

// TestTokenCatalogReconcilesToolchain proves the token catalog joins the
// toolchain's go/token constant set bijectively, with matching spellings and
// lexical classes derived from the toolchain's own predicates.
func TestTokenCatalogReconcilesToolchain(t *testing.T) {
	pkg, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import("go/token")
	if err != nil {
		t.Fatalf("import go/token from source: %v", err)
	}
	catalogByName := map[string]catalog.TokenKind{}
	for _, kind := range catalog.AllTokens() {
		catalogByName[kind.ConstName()] = kind
	}
	toolchain := map[string]token.Token{}
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		constant_, ok := scope.Lookup(name).(*types.Const)
		if !ok || !constant_.Exported() {
			// Unexported constants are the toolchain's internal class
			// boundary markers (literal_beg etc.), not tokens.
			continue
		}
		named, ok := constant_.Type().(*types.Named)
		if !ok || named.Obj().Name() != "Token" {
			continue
		}
		value, exact := constant.Int64Val(constant_.Val())
		if !exact {
			t.Fatalf("token constant %s has non-integer value", name)
		}
		toolchain[name] = token.Token(value)
	}
	if len(toolchain) == 0 {
		t.Fatal("no toolchain token constants found")
	}
	for name, tok := range toolchain {
		kind, cataloged := catalogByName[name]
		if !cataloged {
			t.Errorf("toolchain token %s missing from the catalog", name)
			continue
		}
		if kind.Spelling() != tok.String() {
			t.Errorf("token %s spelling %q, toolchain has %q", name, kind.Spelling(), tok.String())
		}
		var expected catalog.TokenClass
		switch {
		case tok.IsKeyword():
			expected = catalog.TokenClassKeyword
		case tok.IsOperator():
			expected = catalog.TokenClassOperator
		case tok.IsLiteral():
			expected = catalog.TokenClassLiteral
		default:
			expected = catalog.TokenClassSpecial
		}
		if kind.Class() != expected {
			t.Errorf("token %s class %s, toolchain predicates say %s", name, kind.Class(), expected)
		}
	}
	for name := range catalogByName {
		if _, exists := toolchain[name]; !exists {
			t.Errorf("catalog token %s does not exist in the toolchain", name)
		}
	}
}

// TestPredeclaredCatalogReconcilesUniverse proves the predeclared catalog
// joins types.Universe bijectively with matching classes. Debug-build-only
// universe members are tolerated by explicit name.
func TestPredeclaredCatalogReconcilesUniverse(t *testing.T) {
	debugOnly := map[string]bool{"assert": true, "trace": true}
	catalogByName := map[string]catalog.PredeclaredKind{}
	for _, kind := range catalog.AllPredeclared() {
		catalogByName[kind.Name()] = kind
	}
	seen := map[string]bool{}
	for _, name := range types.Universe.Names() {
		if debugOnly[name] {
			continue
		}
		obj := types.Universe.Lookup(name)
		kind, cataloged := catalogByName[name]
		if !cataloged {
			t.Errorf("universe member %s missing from the catalog", name)
			continue
		}
		seen[name] = true
		var expected catalog.PredeclaredClass
		switch obj.(type) {
		case *types.TypeName:
			expected = catalog.PredeclaredClassType
		case *types.Const:
			expected = catalog.PredeclaredClassConstant
		case *types.Nil:
			expected = catalog.PredeclaredClassNil
		case *types.Builtin:
			expected = catalog.PredeclaredClassFunction
		default:
			t.Errorf("universe member %s has unexpected object type %T", name, obj)
			continue
		}
		if kind.Class() != expected {
			t.Errorf("predeclared %s class %s, universe says %s", name, kind.Class(), expected)
		}
	}
	for name := range catalogByName {
		if !seen[name] {
			t.Errorf("catalog predeclared %s does not exist in types.Universe", name)
		}
	}
}

// TestFeatureCatalogWithinSelectedVersion proves every cataloged feature's
// minimum version is well-formed and admitted by the selected Go version.
func TestFeatureCatalogWithinSelectedVersion(t *testing.T) {
	parse := func(v string) (major, minor int) {
		if _, err := fmt.Sscanf(v, "go%d.%d", &major, &minor); err != nil {
			t.Fatalf("unparseable version %q: %v", v, err)
		}
		return major, minor
	}
	selMajor, selMinor := parse(catalog.SelectedGoVersion)
	for _, feature := range catalog.AllFeatures() {
		major, minor := parse(feature.MinVersion())
		if major > selMajor || (major == selMajor && minor > selMinor) {
			t.Errorf("feature %s requires %s beyond selected %s", feature, feature.MinVersion(), catalog.SelectedGoVersion)
		}
	}
}
