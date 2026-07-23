package structure

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// Error is a typed structural-inventory failure.
type Error struct {
	Phase  string
	File   identity.FileID
	Kind   catalog.Kind
	Edge   catalog.Edge
	Span   Span
	Reason string
}

func (e *Error) Error() string {
	return fmt.Sprintf(
		"GOTOTS_STRUCTURE_%s: %s at %s#%d-%d across %s: %s",
		e.Phase, e.Kind, e.File, e.Span.Start.Offset, e.Span.End.Offset, e.Edge, e.Reason,
	)
}

// Classify is the total concrete go/ast-to-catalog binding.
func Classify(node ast.Node) (catalog.Kind, error) {
	switch node.(type) {
	case *ast.BadExpr:
		return catalog.KindBadExpr, nil
	case *ast.Ident:
		return catalog.KindIdent, nil
	case *ast.Ellipsis:
		return catalog.KindEllipsis, nil
	case *ast.BasicLit:
		return catalog.KindBasicLit, nil
	case *ast.FuncLit:
		return catalog.KindFuncLit, nil
	case *ast.CompositeLit:
		return catalog.KindCompositeLit, nil
	case *ast.ParenExpr:
		return catalog.KindParenExpr, nil
	case *ast.SelectorExpr:
		return catalog.KindSelectorExpr, nil
	case *ast.IndexExpr:
		return catalog.KindIndexExpr, nil
	case *ast.IndexListExpr:
		return catalog.KindIndexListExpr, nil
	case *ast.SliceExpr:
		return catalog.KindSliceExpr, nil
	case *ast.TypeAssertExpr:
		return catalog.KindTypeAssertExpr, nil
	case *ast.CallExpr:
		return catalog.KindCallExpr, nil
	case *ast.StarExpr:
		return catalog.KindStarExpr, nil
	case *ast.UnaryExpr:
		return catalog.KindUnaryExpr, nil
	case *ast.BinaryExpr:
		return catalog.KindBinaryExpr, nil
	case *ast.KeyValueExpr:
		return catalog.KindKeyValueExpr, nil
	case *ast.ArrayType:
		return catalog.KindArrayType, nil
	case *ast.StructType:
		return catalog.KindStructType, nil
	case *ast.FuncType:
		return catalog.KindFuncType, nil
	case *ast.InterfaceType:
		return catalog.KindInterfaceType, nil
	case *ast.MapType:
		return catalog.KindMapType, nil
	case *ast.ChanType:
		return catalog.KindChanType, nil
	case *ast.BadStmt:
		return catalog.KindBadStmt, nil
	case *ast.DeclStmt:
		return catalog.KindDeclStmt, nil
	case *ast.EmptyStmt:
		return catalog.KindEmptyStmt, nil
	case *ast.LabeledStmt:
		return catalog.KindLabeledStmt, nil
	case *ast.ExprStmt:
		return catalog.KindExprStmt, nil
	case *ast.SendStmt:
		return catalog.KindSendStmt, nil
	case *ast.IncDecStmt:
		return catalog.KindIncDecStmt, nil
	case *ast.AssignStmt:
		return catalog.KindAssignStmt, nil
	case *ast.GoStmt:
		return catalog.KindGoStmt, nil
	case *ast.DeferStmt:
		return catalog.KindDeferStmt, nil
	case *ast.ReturnStmt:
		return catalog.KindReturnStmt, nil
	case *ast.BranchStmt:
		return catalog.KindBranchStmt, nil
	case *ast.BlockStmt:
		return catalog.KindBlockStmt, nil
	case *ast.IfStmt:
		return catalog.KindIfStmt, nil
	case *ast.CaseClause:
		return catalog.KindCaseClause, nil
	case *ast.SwitchStmt:
		return catalog.KindSwitchStmt, nil
	case *ast.TypeSwitchStmt:
		return catalog.KindTypeSwitchStmt, nil
	case *ast.CommClause:
		return catalog.KindCommClause, nil
	case *ast.SelectStmt:
		return catalog.KindSelectStmt, nil
	case *ast.ForStmt:
		return catalog.KindForStmt, nil
	case *ast.RangeStmt:
		return catalog.KindRangeStmt, nil
	case *ast.BadDecl:
		return catalog.KindBadDecl, nil
	case *ast.GenDecl:
		return catalog.KindGenDecl, nil
	case *ast.FuncDecl:
		return catalog.KindFuncDecl, nil
	case *ast.ImportSpec:
		return catalog.KindImportSpec, nil
	case *ast.ValueSpec:
		return catalog.KindValueSpec, nil
	case *ast.TypeSpec:
		return catalog.KindTypeSpec, nil
	case *ast.File:
		return catalog.KindFile, nil
	case *ast.Comment:
		return catalog.KindComment, nil
	case *ast.CommentGroup:
		return catalog.KindCommentGroup, nil
	case *ast.Field:
		return catalog.KindField, nil
	case *ast.FieldList:
		return catalog.KindFieldList, nil
	case *ast.Directive:
		return catalog.KindDirective, nil
	case *ast.Package:
		return catalog.KindPackage, nil
	default:
		return catalog.KindInvalid, &Error{Phase: "UNKNOWN_CONSTRUCT", Reason: fmt.Sprintf("no catalog kind for %T", node)}
	}
}

// Child is one transient cataloged syntax edge.
type Child struct {
	node    ast.Node
	edge    catalog.Edge
	ordinal int
}

func (c Child) Node() ast.Node     { return c.node }
func (c Child) Edge() catalog.Edge { return c.edge }
func (c Child) Ordinal() int       { return c.ordinal }

// children returns exactly the cataloged child edges in pinned order. It owns
// no private syntax-edge table.
func children(node ast.Node, kind catalog.Kind) ([]Child, error) {
	value := reflect.ValueOf(node)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return nil, fmt.Errorf("catalog node %T is not a non-nil pointer", node)
	}
	value = value.Elem()
	var out []Child
	for _, edge := range catalog.EdgesOf(kind) {
		field := value.FieldByName(edge.Field())
		if !field.IsValid() {
			return nil, fmt.Errorf("catalog edge %s names no field on %T", edge, node)
		}
		if edge.IsList() {
			if field.Kind() != reflect.Slice {
				return nil, fmt.Errorf("catalog list edge %s is not a slice on %T", edge, node)
			}
			for index := 0; index < field.Len(); index++ {
				item := field.Index(index)
				if (item.Kind() == reflect.Interface ||
					item.Kind() == reflect.Pointer) &&
					!item.IsNil() {
					if nested, ok := item.Interface().(ast.Node); ok {
						out = append(out, Child{node: nested, edge: edge, ordinal: index})
					}
				}
			}
			continue
		}
		if (field.Kind() == reflect.Interface || field.Kind() == reflect.Pointer) && !field.IsNil() {
			if nested, ok := field.Interface().(ast.Node); ok {
				out = append(out, Child{node: nested, edge: edge})
			}
		}
	}
	return out, nil
}

// Children returns exactly the cataloged children in pinned order.
func Children(node ast.Node, kind catalog.Kind) ([]Child, error) {
	return children(node, kind)
}

// TokenEvidence returns the lexical token catalog identity of a token-bearing
// construct.
func TokenEvidence(node ast.Node) (catalog.TokenKind, error) {
	return tokenEvidence(node)
}

func tokenEvidence(node ast.Node) (catalog.TokenKind, error) {
	var lexical token.Token
	switch node := node.(type) {
	case *ast.BinaryExpr:
		lexical = node.Op
	case *ast.UnaryExpr:
		lexical = node.Op
	case *ast.AssignStmt:
		lexical = node.Tok
	case *ast.IncDecStmt:
		lexical = node.Tok
	case *ast.BranchStmt:
		lexical = node.Tok
	case *ast.GenDecl:
		lexical = node.Tok
	case *ast.BasicLit:
		lexical = node.Kind
	default:
		return catalog.TokenInvalid, nil
	}
	bound := catalog.TokenBySpelling(lexical.String())
	if !bound.Valid() {
		return catalog.TokenInvalid, fmt.Errorf("token %q is absent from the catalog", lexical)
	}
	return bound, nil
}
