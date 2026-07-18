package census

import (
	"fmt"
	"go/ast"
)

// astKindName names every AST node kind the pinned toolchain grammar can
// produce. An unknown kind fails the census: it means the grammar has a
// construct this tool has never been reviewed against.
func astKindName(node ast.Node) (string, error) {
	switch node.(type) {
	case *ast.File:
		return "File", nil
	case *ast.Comment:
		return "Comment", nil
	case *ast.CommentGroup:
		return "CommentGroup", nil
	case *ast.Field:
		return "Field", nil
	case *ast.FieldList:
		return "FieldList", nil
	case *ast.Ident:
		return "Ident", nil
	case *ast.Ellipsis:
		return "Ellipsis", nil
	case *ast.BasicLit:
		return "BasicLit", nil
	case *ast.FuncLit:
		return "FuncLit", nil
	case *ast.CompositeLit:
		return "CompositeLit", nil
	case *ast.ParenExpr:
		return "ParenExpr", nil
	case *ast.SelectorExpr:
		return "SelectorExpr", nil
	case *ast.IndexExpr:
		return "IndexExpr", nil
	case *ast.IndexListExpr:
		return "IndexListExpr", nil
	case *ast.SliceExpr:
		return "SliceExpr", nil
	case *ast.TypeAssertExpr:
		return "TypeAssertExpr", nil
	case *ast.CallExpr:
		return "CallExpr", nil
	case *ast.StarExpr:
		return "StarExpr", nil
	case *ast.UnaryExpr:
		return "UnaryExpr", nil
	case *ast.BinaryExpr:
		return "BinaryExpr", nil
	case *ast.KeyValueExpr:
		return "KeyValueExpr", nil
	case *ast.ArrayType:
		return "ArrayType", nil
	case *ast.StructType:
		return "StructType", nil
	case *ast.FuncType:
		return "FuncType", nil
	case *ast.InterfaceType:
		return "InterfaceType", nil
	case *ast.MapType:
		return "MapType", nil
	case *ast.ChanType:
		return "ChanType", nil
	case *ast.DeclStmt:
		return "DeclStmt", nil
	case *ast.EmptyStmt:
		return "EmptyStmt", nil
	case *ast.LabeledStmt:
		return "LabeledStmt", nil
	case *ast.ExprStmt:
		return "ExprStmt", nil
	case *ast.SendStmt:
		return "SendStmt", nil
	case *ast.IncDecStmt:
		return "IncDecStmt", nil
	case *ast.AssignStmt:
		return "AssignStmt", nil
	case *ast.GoStmt:
		return "GoStmt", nil
	case *ast.DeferStmt:
		return "DeferStmt", nil
	case *ast.ReturnStmt:
		return "ReturnStmt", nil
	case *ast.BranchStmt:
		return "BranchStmt", nil
	case *ast.BlockStmt:
		return "BlockStmt", nil
	case *ast.IfStmt:
		return "IfStmt", nil
	case *ast.CaseClause:
		return "CaseClause", nil
	case *ast.SwitchStmt:
		return "SwitchStmt", nil
	case *ast.TypeSwitchStmt:
		return "TypeSwitchStmt", nil
	case *ast.CommClause:
		return "CommClause", nil
	case *ast.SelectStmt:
		return "SelectStmt", nil
	case *ast.ForStmt:
		return "ForStmt", nil
	case *ast.RangeStmt:
		return "RangeStmt", nil
	case *ast.GenDecl:
		return "GenDecl", nil
	case *ast.FuncDecl:
		return "FuncDecl", nil
	case *ast.ImportSpec:
		return "ImportSpec", nil
	case *ast.ValueSpec:
		return "ValueSpec", nil
	case *ast.TypeSpec:
		return "TypeSpec", nil
	case *ast.BadExpr, *ast.BadStmt, *ast.BadDecl:
		return "", fmt.Errorf("parse produced a Bad node at an analyzed position; refusing to count a broken file")
	default:
		return "", fmt.Errorf("unknown AST node kind %T: the pinned toolchain grammar has a construct this census has not been reviewed against", node)
	}
}
