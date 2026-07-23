package stagecheck

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

var independentKindByName = func() map[string]catalog.Kind {
	out := map[string]catalog.Kind{}
	for _, kind := range catalog.All() {
		out[kind.Name()] = kind
	}
	return out
}()

func independentDefinitionKind(
	node ast.Node,
	context catalog.DefinitionContext,
) (identity.DefinitionKind, bool, error) {
	kind, err := independentKind(node)
	if err != nil {
		return identity.DefinitionInvalid, false, err
	}
	entries, err := independentDefinitionEntries(node)
	if err != nil {
		return identity.DefinitionInvalid, false, err
	}
	return catalog.DefinitionKind(kind, context, len(entries) > 0)
}

func independentDefinitionEntries(
	node ast.Node,
) ([]derivedChild, error) {
	kind, err := independentKind(node)
	if err != nil {
		return nil, err
	}
	children, err := independentChildren(node, kind)
	if err != nil {
		return nil, err
	}
	var entries []derivedChild
	for _, child := range children {
		if child.edge.DefinitionEntry() {
			entries = append(entries, child)
		}
	}
	return entries, nil
}

func independentChildContext(
	node ast.Node,
	kind catalog.Kind,
	context catalog.DefinitionContext,
) (catalog.DefinitionContext, error) {
	if kind != catalog.KindGenDecl {
		return context, nil
	}
	declaration, err := independentToken(node)
	if err != nil {
		return catalog.DefinitionContext{}, err
	}
	return context.WithDeclaration(declaration)
}

func independentKind(node ast.Node) (catalog.Kind, error) {
	if node == nil {
		return catalog.KindInvalid, fmt.Errorf("nil AST node")
	}
	typ := reflect.TypeOf(node)
	if typ.Kind() != reflect.Pointer {
		return catalog.KindInvalid, fmt.Errorf(
			"non-pointer AST node %T", node,
		)
	}
	kind := independentKindByName[typ.Elem().Name()]
	if !kind.Valid() {
		return catalog.KindInvalid, fmt.Errorf(
			"independent classifier has no catalog kind for %T", node,
		)
	}
	return kind, nil
}

func independentChildren(
	node ast.Node,
	kind catalog.Kind,
) ([]derivedChild, error) {
	value := reflect.ValueOf(node)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return nil, fmt.Errorf(
			"invalid independent AST node %T", node,
		)
	}
	value = value.Elem()
	typ := value.Type()
	edgeByField := map[string]catalog.Edge{}
	for _, edge := range catalog.EdgesOf(kind) {
		edgeByField[edge.Field()] = edge
	}
	excluded := map[string]bool{}
	for _, field := range catalog.ExcludedFields() {
		if field.Kind == kind {
			excluded[field.Field] = true
		}
	}
	var out []derivedChild
	for fieldIndex := 0; fieldIndex < value.NumField(); fieldIndex++ {
		fieldName := typ.Field(fieldIndex).Name
		field := value.Field(fieldIndex)
		var nodes []ast.Node
		collectIndependentNodes(field, &nodes)
		if len(nodes) == 0 {
			continue
		}
		edge, cataloged := edgeByField[fieldName]
		if !cataloged {
			if excluded[fieldName] {
				continue
			}
			return nil, fmt.Errorf(
				"independent walk found uncataloged %s.%s",
				kind, fieldName,
			)
		}
		for ordinal, child := range nodes {
			out = append(out, derivedChild{
				node: child, edge: edge, ordinal: ordinal,
			})
		}
	}
	return out, nil
}

func collectIndependentNodes(
	value reflect.Value,
	out *[]ast.Node,
) {
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			return
		}
		if node, ok := value.Interface().(ast.Node); ok {
			*out = append(*out, node)
		}
	case reflect.Slice:
		for index := 0; index < value.Len(); index++ {
			collectIndependentNodes(value.Index(index), out)
		}
	}
}

func independentToken(node ast.Node) (catalog.TokenKind, error) {
	var lexical token.Token
	switch typed := node.(type) {
	case *ast.BinaryExpr:
		lexical = typed.Op
	case *ast.UnaryExpr:
		lexical = typed.Op
	case *ast.AssignStmt:
		lexical = typed.Tok
	case *ast.IncDecStmt:
		lexical = typed.Tok
	case *ast.BranchStmt:
		lexical = typed.Tok
	case *ast.GenDecl:
		lexical = typed.Tok
	case *ast.BasicLit:
		lexical = typed.Kind
	default:
		return catalog.TokenInvalid, nil
	}
	bound := catalog.TokenBySpelling(lexical.String())
	if !bound.Valid() {
		return catalog.TokenInvalid, fmt.Errorf(
			"independent token %q is not cataloged", lexical,
		)
	}
	return bound, nil
}

func independentDefinitionName(node ast.Node) string {
	switch typed := node.(type) {
	case *ast.FuncDecl:
		return typed.Name.Name
	case *ast.FuncLit:
		return "func literal"
	case *ast.ValueSpec:
		names := make([]string, 0, len(typed.Names))
		for _, name := range typed.Names {
			names = append(names, name.Name)
		}
		return strings.Join(names, ",")
	default:
		return ""
	}
}
