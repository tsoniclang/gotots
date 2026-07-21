// Package analyze binds concrete Go syntax to the closed construct catalog and
// (in later phases) to the target-independent semantic model. With
// internal/source it is one of two packages permitted to import go/ast and
// go/types; it never leaks raw AST nodes to downstream layers.
package analyze

import (
	"fmt"
	"go/ast"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// UnknownConstructError reports a concrete syntax node with no catalog Kind.
// Classification fails closed: an unrecognized form is a typed error, never a
// default or best-effort classification.
type UnknownConstructError struct {
	// GoType is the concrete go/ast node type that has no catalog identity.
	GoType string
}

func (e *UnknownConstructError) Error() string {
	return fmt.Sprintf("GOTOTS_UNKNOWN_CONSTRUCT: no catalog kind for go/ast node %s", e.GoType)
}

// Classify maps one concrete go/ast node to its catalog Kind. It is total over
// every concrete type implementing ast.Node in the selected Go toolchain
// (proven by TestClassifyIsToolchainBijection) and fails closed on any
// unrecognized node with an UnknownConstructError. There is no default
// classification and no textual fallback.
//
// ast.Package is deprecated and ast.Directive is produced only by
// ParseDirective, so neither appears in a file-parse tree; both are classified
// for total reconciliation and carry their catalog disposition.
func Classify(n ast.Node) (catalog.Kind, error) {
	switch n.(type) {
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
	}
	return catalog.KindInvalid, &UnknownConstructError{GoType: fmt.Sprintf("%T", n)}
}
