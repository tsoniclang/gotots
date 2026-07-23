package structure

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"sort"
	"testing"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

type nodeCase struct {
	node ast.Node
	kind catalog.Kind
}

type unknownNode struct{}

func (unknownNode) Pos() token.Pos { return 1 }
func (unknownNode) End() token.Pos { return 2 }

var nodeCases = []nodeCase{
	{(*ast.BadExpr)(nil), catalog.KindBadExpr},
	{(*ast.Ident)(nil), catalog.KindIdent},
	{(*ast.Ellipsis)(nil), catalog.KindEllipsis},
	{(*ast.BasicLit)(nil), catalog.KindBasicLit},
	{(*ast.FuncLit)(nil), catalog.KindFuncLit},
	{(*ast.CompositeLit)(nil), catalog.KindCompositeLit},
	{(*ast.ParenExpr)(nil), catalog.KindParenExpr},
	{(*ast.SelectorExpr)(nil), catalog.KindSelectorExpr},
	{(*ast.IndexExpr)(nil), catalog.KindIndexExpr},
	{(*ast.IndexListExpr)(nil), catalog.KindIndexListExpr},
	{(*ast.SliceExpr)(nil), catalog.KindSliceExpr},
	{(*ast.TypeAssertExpr)(nil), catalog.KindTypeAssertExpr},
	{(*ast.CallExpr)(nil), catalog.KindCallExpr},
	{(*ast.StarExpr)(nil), catalog.KindStarExpr},
	{(*ast.UnaryExpr)(nil), catalog.KindUnaryExpr},
	{(*ast.BinaryExpr)(nil), catalog.KindBinaryExpr},
	{(*ast.KeyValueExpr)(nil), catalog.KindKeyValueExpr},
	{(*ast.ArrayType)(nil), catalog.KindArrayType},
	{(*ast.StructType)(nil), catalog.KindStructType},
	{(*ast.FuncType)(nil), catalog.KindFuncType},
	{(*ast.InterfaceType)(nil), catalog.KindInterfaceType},
	{(*ast.MapType)(nil), catalog.KindMapType},
	{(*ast.ChanType)(nil), catalog.KindChanType},
	{(*ast.BadStmt)(nil), catalog.KindBadStmt},
	{(*ast.DeclStmt)(nil), catalog.KindDeclStmt},
	{(*ast.EmptyStmt)(nil), catalog.KindEmptyStmt},
	{(*ast.LabeledStmt)(nil), catalog.KindLabeledStmt},
	{(*ast.ExprStmt)(nil), catalog.KindExprStmt},
	{(*ast.SendStmt)(nil), catalog.KindSendStmt},
	{(*ast.IncDecStmt)(nil), catalog.KindIncDecStmt},
	{(*ast.AssignStmt)(nil), catalog.KindAssignStmt},
	{(*ast.GoStmt)(nil), catalog.KindGoStmt},
	{(*ast.DeferStmt)(nil), catalog.KindDeferStmt},
	{(*ast.ReturnStmt)(nil), catalog.KindReturnStmt},
	{(*ast.BranchStmt)(nil), catalog.KindBranchStmt},
	{(*ast.BlockStmt)(nil), catalog.KindBlockStmt},
	{(*ast.IfStmt)(nil), catalog.KindIfStmt},
	{(*ast.CaseClause)(nil), catalog.KindCaseClause},
	{(*ast.SwitchStmt)(nil), catalog.KindSwitchStmt},
	{(*ast.TypeSwitchStmt)(nil), catalog.KindTypeSwitchStmt},
	{(*ast.CommClause)(nil), catalog.KindCommClause},
	{(*ast.SelectStmt)(nil), catalog.KindSelectStmt},
	{(*ast.ForStmt)(nil), catalog.KindForStmt},
	{(*ast.RangeStmt)(nil), catalog.KindRangeStmt},
	{(*ast.BadDecl)(nil), catalog.KindBadDecl},
	{(*ast.GenDecl)(nil), catalog.KindGenDecl},
	{(*ast.FuncDecl)(nil), catalog.KindFuncDecl},
	{(*ast.ImportSpec)(nil), catalog.KindImportSpec},
	{(*ast.ValueSpec)(nil), catalog.KindValueSpec},
	{(*ast.TypeSpec)(nil), catalog.KindTypeSpec},
	{(*ast.File)(nil), catalog.KindFile},
	{(*ast.Comment)(nil), catalog.KindComment},
	{(*ast.CommentGroup)(nil), catalog.KindCommentGroup},
	{(*ast.Field)(nil), catalog.KindField},
	{(*ast.FieldList)(nil), catalog.KindFieldList},
	{(*ast.Directive)(nil), catalog.KindDirective},
	{(*ast.Package)(nil), catalog.KindPackage},
}

func TestClassifierExactJoinsToolchainNodeUniverse(t *testing.T) {
	toolchain := toolchainNodeNames(t)
	cases := map[string]catalog.Kind{}
	kinds := map[catalog.Kind]string{}
	for _, test := range nodeCases {
		name := reflect.TypeOf(test.node).Elem().Name()
		if _, duplicate := cases[name]; duplicate {
			t.Fatalf("node case %s is duplicated", name)
		}
		if prior, duplicate := kinds[test.kind]; duplicate {
			t.Fatalf("kind %s binds both %s and %s", test.kind, prior, name)
		}
		cases[name] = test.kind
		kinds[test.kind] = name
		got, err := Classify(test.node)
		if err != nil {
			t.Errorf("Classify((*ast.%s)(nil)): %v", name, err)
		} else if got != test.kind {
			t.Errorf("Classify((*ast.%s)(nil)) = %s, want %s", name, got, test.kind)
		}
	}
	joinStringSets(t, "toolchain node", toolchain, "classifier case", cases)
	if len(kinds) != len(catalog.All()) {
		t.Fatalf("classifier covers %d catalog kinds, catalog has %d", len(kinds), len(catalog.All()))
	}
	for _, kind := range catalog.All() {
		if _, present := kinds[kind]; !present {
			t.Errorf("catalog kind %s has no classifier case", kind)
		}
	}
}

func TestClassifierRejectsInjectedUnknownNode(t *testing.T) {
	if kind, err := Classify(unknownNode{}); err == nil || kind.Valid() {
		t.Fatalf(
			"injected unknown node classified as %s with error %v",
			kind,
			err,
		)
	}
}

func TestCatalogEdgesExactJoinToolchainNodeFields(t *testing.T) {
	nodeType := reflect.TypeOf((*ast.Node)(nil)).Elem()
	excluded := map[catalog.Kind]map[string]bool{}
	for _, record := range catalog.ExcludedFields() {
		if record.Reason == "" {
			t.Errorf("excluded field %s.%s has no reason", record.Kind, record.Field)
		}
		if excluded[record.Kind] == nil {
			excluded[record.Kind] = map[string]bool{}
		}
		if excluded[record.Kind][record.Field] {
			t.Errorf("excluded field %s.%s is duplicated", record.Kind, record.Field)
		}
		excluded[record.Kind][record.Field] = true
	}

	for _, test := range nodeCases {
		structType := reflect.TypeOf(test.node).Elem()
		var actualNames []string
		actualLists := map[string]bool{}
		nodeFields := map[string]bool{}
		for index := 0; index < structType.NumField(); index++ {
			field := structType.Field(index)
			if field.PkgPath != "" {
				continue
			}
			bearing, list := nodeBearing(field.Type, nodeType)
			if !bearing {
				continue
			}
			nodeFields[field.Name] = true
			if excluded[test.kind][field.Name] {
				continue
			}
			actualNames = append(actualNames, field.Name)
			actualLists[field.Name] = list
		}
		for field := range excluded[test.kind] {
			if !nodeFields[field] {
				t.Errorf("exclusion %s.%s does not name a node-bearing toolchain field", test.kind, field)
			}
		}

		edges := catalog.EdgesOf(test.kind)
		if len(edges) != len(actualNames) {
			t.Errorf("%s has %d catalog edges, toolchain walk has %d: %v", test.kind, len(edges), len(actualNames), actualNames)
			continue
		}
		for index, edge := range edges {
			if edge.Parent() != test.kind {
				t.Errorf("edge %s parent = %s, want %s", edge, edge.Parent(), test.kind)
			}
			if edge.Field() != actualNames[index] {
				t.Errorf("%s edge %d field = %s, toolchain order has %s", test.kind, index, edge.Field(), actualNames[index])
			}
			if edge.IsList() != actualLists[edge.Field()] {
				t.Errorf("edge %s list=%t, toolchain collection=%t", edge, edge.IsList(), actualLists[edge.Field()])
			}
			if edge.Name() != test.kind.Name()+"."+edge.Field() {
				t.Errorf("edge %s has non-canonical name %q", edge, edge.Name())
			}
		}
	}
}

func TestChildrenExactJoinToolchainWalk(t *testing.T) {
	source := `package fixture

import "fmt"

type Box[T any] struct { Value T }

func outer[T ~int](input T) func(int) int {
	var values = []int{1, 2, 3}
	return func(delta int) int {
		for index, value := range values {
			if index > 0 {
				values[index] = value + delta
			}
		}
		fmt.Println(values)
		return int(input) + values[0]
	}
}
`
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			return true
		}
		kind, err := Classify(node)
		if err != nil {
			t.Errorf("classify %T: %v", node, err)
			return false
		}
		got, err := Children(node, kind)
		if err != nil {
			t.Errorf("children %T: %v", node, err)
			return false
		}
		want := directToolchainChildren(node)
		if len(got) != len(want) {
			t.Errorf("%T has %d catalog children, toolchain walk has %d", node, len(got), len(want))
			return true
		}
		for index := range want {
			if got[index].Node() != want[index] {
				t.Errorf("%T child %d = %T, toolchain walk has %T", node, index, got[index].Node(), want[index])
			}
		}
		return true
	})
}

func toolchainNodeNames(t *testing.T) map[string]catalog.Kind {
	t.Helper()
	pkg, err := importer.Default().Import("go/ast")
	if err != nil {
		t.Fatal(err)
	}
	nodeObject := pkg.Scope().Lookup("Node")
	node, ok := nodeObject.Type().Underlying().(*types.Interface)
	if !ok {
		t.Fatalf("go/ast.Node has type %T", nodeObject.Type().Underlying())
	}
	node.Complete()
	out := map[string]catalog.Kind{}
	for _, name := range pkg.Scope().Names() {
		object, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := object.Type().(*types.Named)
		if !ok {
			continue
		}
		if _, ok := named.Underlying().(*types.Struct); ok &&
			types.Implements(types.NewPointer(named), node) {
			out[name] = catalog.KindInvalid
		}
	}
	return out
}

func joinStringSets(
	t *testing.T,
	leftName string,
	left map[string]catalog.Kind,
	rightName string,
	right map[string]catalog.Kind,
) {
	t.Helper()
	var onlyLeft, onlyRight []string
	for name := range left {
		if _, present := right[name]; !present {
			onlyLeft = append(onlyLeft, name)
		}
	}
	for name := range right {
		if _, present := left[name]; !present {
			onlyRight = append(onlyRight, name)
		}
	}
	sort.Strings(onlyLeft)
	sort.Strings(onlyRight)
	if len(onlyLeft) != 0 || len(onlyRight) != 0 {
		t.Fatalf("%s only=%v; %s only=%v", leftName, onlyLeft, rightName, onlyRight)
	}
}

func nodeBearing(field reflect.Type, node reflect.Type) (bool, bool) {
	if field.Implements(node) {
		return true, false
	}
	switch field.Kind() {
	case reflect.Slice, reflect.Array:
		return field.Elem().Implements(node), true
	case reflect.Map:
		return field.Elem().Implements(node), true
	default:
		return false, false
	}
}

func directToolchainChildren(root ast.Node) []ast.Node {
	var stack []ast.Node
	var out []ast.Node
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) == 1 {
			out = append(out, node)
		}
		stack = append(stack, node)
		return true
	})
	return out
}
