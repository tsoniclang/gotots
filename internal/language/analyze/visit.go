package analyze

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/source"
)

// BuildInventory produces the authoritative construct inventory of a loaded
// source file using a parent-directed recursive descent: each parent dispatches
// on its own concrete type and visits every child edge, passing itself as the
// parent. A child never inspects its parent to recover meaning. A
// classification failure aborts the inventory (fail closed).
func BuildInventory(src *source.File) (Inventory, error) {
	v := &visitor{fset: src.Fset}
	if _, err := v.occurrence(src.Syntax, ""); err != nil {
		return Inventory{}, err
	}
	return Inventory{Path: src.Path, Occurrences: v.occurrences}, nil
}

type visitor struct {
	fset        *token.FileSet
	occurrences []Occurrence
}

// occurrence records n with parent as its enclosing occurrence, then descends
// into n's children with n as their parent.
func (v *visitor) occurrence(n ast.Node, parent OccurrenceID) (OccurrenceID, error) {
	kind, err := Classify(n)
	if err != nil {
		return "", err
	}
	span := v.spanOf(n)
	id := OccurrenceID(fmt.Sprintf("%s#%d:%d/%s", span.Filename, span.Start.Offset, span.End.Offset, kind.Name()))
	v.occurrences = append(v.occurrences, Occurrence{ID: id, Kind: kind, Parent: parent, Span: span})
	if err := v.children(n, id); err != nil {
		return "", err
	}
	return id, nil
}

func (v *visitor) spanOf(n ast.Node) Span {
	start := v.fset.Position(n.Pos())
	end := v.fset.Position(n.End())
	return Span{
		Filename: start.Filename,
		Start:    Position{Line: start.Line, Column: start.Column, Offset: start.Offset},
		End:      Position{Line: end.Line, Column: end.Column, Offset: end.Offset},
	}
}

// node descends into one required child.
func (v *visitor) node(n ast.Node, parent OccurrenceID) error {
	_, err := v.occurrence(n, parent)
	return err
}

func (v *visitor) exprs(list []ast.Expr, parent OccurrenceID) error {
	for _, x := range list {
		if err := v.node(x, parent); err != nil {
			return err
		}
	}
	return nil
}

func (v *visitor) stmts(list []ast.Stmt, parent OccurrenceID) error {
	for _, s := range list {
		if err := v.node(s, parent); err != nil {
			return err
		}
	}
	return nil
}

func (v *visitor) idents(list []*ast.Ident, parent OccurrenceID) error {
	for _, ident := range list {
		if err := v.node(ident, parent); err != nil {
			return err
		}
	}
	return nil
}

func (v *visitor) specs(list []ast.Spec, parent OccurrenceID) error {
	for _, spec := range list {
		if err := v.node(spec, parent); err != nil {
			return err
		}
	}
	return nil
}

func (v *visitor) decls(list []ast.Decl, parent OccurrenceID) error {
	for _, decl := range list {
		if err := v.node(decl, parent); err != nil {
			return err
		}
	}
	return nil
}

func (v *visitor) fields(list []*ast.Field, parent OccurrenceID) error {
	for _, field := range list {
		if err := v.node(field, parent); err != nil {
			return err
		}
	}
	return nil
}

func (v *visitor) comments(list []*ast.Comment, parent OccurrenceID) error {
	for _, comment := range list {
		if err := v.node(comment, parent); err != nil {
			return err
		}
	}
	return nil
}

// children visits every child edge of n in source order. It mirrors the
// toolchain's own child enumeration so the inventory covers exactly the nodes a
// standard walk reaches; a type reaching the default arm is a fail-closed
// coverage defect.
func (v *visitor) children(n ast.Node, id OccurrenceID) error {
	switch n := n.(type) {
	case *ast.Comment, *ast.BadExpr, *ast.Ident, *ast.BasicLit, *ast.BadStmt,
		*ast.EmptyStmt, *ast.BadDecl, *ast.Directive:
		return nil // leaves
	case *ast.CommentGroup:
		return v.comments(n.List, id)
	case *ast.Field:
		if err := v.optional(n.Doc, id); err != nil {
			return err
		}
		if err := v.idents(n.Names, id); err != nil {
			return err
		}
		if err := v.optionalExpr(n.Type, id); err != nil {
			return err
		}
		if err := v.optionalBasicLit(n.Tag, id); err != nil {
			return err
		}
		return v.optional(n.Comment, id)
	case *ast.FieldList:
		return v.fields(n.List, id)
	case *ast.Ellipsis:
		return v.optionalExpr(n.Elt, id)
	case *ast.FuncLit:
		if err := v.node(n.Type, id); err != nil {
			return err
		}
		return v.node(n.Body, id)
	case *ast.CompositeLit:
		if err := v.optionalExpr(n.Type, id); err != nil {
			return err
		}
		return v.exprs(n.Elts, id)
	case *ast.ParenExpr:
		return v.node(n.X, id)
	case *ast.SelectorExpr:
		if err := v.node(n.X, id); err != nil {
			return err
		}
		return v.node(n.Sel, id)
	case *ast.IndexExpr:
		if err := v.node(n.X, id); err != nil {
			return err
		}
		return v.node(n.Index, id)
	case *ast.IndexListExpr:
		if err := v.node(n.X, id); err != nil {
			return err
		}
		return v.exprs(n.Indices, id)
	case *ast.SliceExpr:
		if err := v.node(n.X, id); err != nil {
			return err
		}
		if err := v.optionalExpr(n.Low, id); err != nil {
			return err
		}
		if err := v.optionalExpr(n.High, id); err != nil {
			return err
		}
		return v.optionalExpr(n.Max, id)
	case *ast.TypeAssertExpr:
		if err := v.node(n.X, id); err != nil {
			return err
		}
		return v.optionalExpr(n.Type, id)
	case *ast.CallExpr:
		if err := v.node(n.Fun, id); err != nil {
			return err
		}
		return v.exprs(n.Args, id)
	case *ast.StarExpr:
		return v.node(n.X, id)
	case *ast.UnaryExpr:
		return v.node(n.X, id)
	case *ast.BinaryExpr:
		if err := v.node(n.X, id); err != nil {
			return err
		}
		return v.node(n.Y, id)
	case *ast.KeyValueExpr:
		if err := v.node(n.Key, id); err != nil {
			return err
		}
		return v.node(n.Value, id)
	case *ast.ArrayType:
		if err := v.optionalExpr(n.Len, id); err != nil {
			return err
		}
		return v.node(n.Elt, id)
	case *ast.StructType:
		return v.node(n.Fields, id)
	case *ast.FuncType:
		if err := v.optionalFieldList(n.TypeParams, id); err != nil {
			return err
		}
		if err := v.optionalFieldList(n.Params, id); err != nil {
			return err
		}
		return v.optionalFieldList(n.Results, id)
	case *ast.InterfaceType:
		return v.node(n.Methods, id)
	case *ast.MapType:
		if err := v.node(n.Key, id); err != nil {
			return err
		}
		return v.node(n.Value, id)
	case *ast.ChanType:
		return v.node(n.Value, id)
	case *ast.DeclStmt:
		return v.node(n.Decl, id)
	case *ast.LabeledStmt:
		if err := v.node(n.Label, id); err != nil {
			return err
		}
		return v.node(n.Stmt, id)
	case *ast.ExprStmt:
		return v.node(n.X, id)
	case *ast.SendStmt:
		if err := v.node(n.Chan, id); err != nil {
			return err
		}
		return v.node(n.Value, id)
	case *ast.IncDecStmt:
		return v.node(n.X, id)
	case *ast.AssignStmt:
		if err := v.exprs(n.Lhs, id); err != nil {
			return err
		}
		return v.exprs(n.Rhs, id)
	case *ast.GoStmt:
		return v.node(n.Call, id)
	case *ast.DeferStmt:
		return v.node(n.Call, id)
	case *ast.ReturnStmt:
		return v.exprs(n.Results, id)
	case *ast.BranchStmt:
		return v.optionalIdent(n.Label, id)
	case *ast.BlockStmt:
		return v.stmts(n.List, id)
	case *ast.IfStmt:
		if err := v.optionalStmt(n.Init, id); err != nil {
			return err
		}
		if err := v.node(n.Cond, id); err != nil {
			return err
		}
		if err := v.node(n.Body, id); err != nil {
			return err
		}
		return v.optionalStmt(n.Else, id)
	case *ast.CaseClause:
		if err := v.exprs(n.List, id); err != nil {
			return err
		}
		return v.stmts(n.Body, id)
	case *ast.SwitchStmt:
		if err := v.optionalStmt(n.Init, id); err != nil {
			return err
		}
		if err := v.optionalExpr(n.Tag, id); err != nil {
			return err
		}
		return v.node(n.Body, id)
	case *ast.TypeSwitchStmt:
		if err := v.optionalStmt(n.Init, id); err != nil {
			return err
		}
		if err := v.node(n.Assign, id); err != nil {
			return err
		}
		return v.node(n.Body, id)
	case *ast.CommClause:
		if err := v.optionalStmt(n.Comm, id); err != nil {
			return err
		}
		return v.stmts(n.Body, id)
	case *ast.SelectStmt:
		return v.node(n.Body, id)
	case *ast.ForStmt:
		if err := v.optionalStmt(n.Init, id); err != nil {
			return err
		}
		if err := v.optionalExpr(n.Cond, id); err != nil {
			return err
		}
		if err := v.optionalStmt(n.Post, id); err != nil {
			return err
		}
		return v.node(n.Body, id)
	case *ast.RangeStmt:
		if err := v.optionalExpr(n.Key, id); err != nil {
			return err
		}
		if err := v.optionalExpr(n.Value, id); err != nil {
			return err
		}
		if err := v.node(n.X, id); err != nil {
			return err
		}
		return v.node(n.Body, id)
	case *ast.ImportSpec:
		if err := v.optional(n.Doc, id); err != nil {
			return err
		}
		if err := v.optionalIdent(n.Name, id); err != nil {
			return err
		}
		if err := v.node(n.Path, id); err != nil {
			return err
		}
		return v.optional(n.Comment, id)
	case *ast.ValueSpec:
		if err := v.optional(n.Doc, id); err != nil {
			return err
		}
		if err := v.idents(n.Names, id); err != nil {
			return err
		}
		if err := v.optionalExpr(n.Type, id); err != nil {
			return err
		}
		if err := v.exprs(n.Values, id); err != nil {
			return err
		}
		return v.optional(n.Comment, id)
	case *ast.TypeSpec:
		if err := v.optional(n.Doc, id); err != nil {
			return err
		}
		if err := v.node(n.Name, id); err != nil {
			return err
		}
		if err := v.optionalFieldList(n.TypeParams, id); err != nil {
			return err
		}
		if err := v.node(n.Type, id); err != nil {
			return err
		}
		return v.optional(n.Comment, id)
	case *ast.GenDecl:
		if err := v.optional(n.Doc, id); err != nil {
			return err
		}
		return v.specs(n.Specs, id)
	case *ast.FuncDecl:
		if err := v.optional(n.Doc, id); err != nil {
			return err
		}
		if err := v.optionalFieldList(n.Recv, id); err != nil {
			return err
		}
		if err := v.node(n.Name, id); err != nil {
			return err
		}
		if err := v.node(n.Type, id); err != nil {
			return err
		}
		return v.optionalStmt(n.Body, id)
	case *ast.File:
		if err := v.optional(n.Doc, id); err != nil {
			return err
		}
		if err := v.node(n.Name, id); err != nil {
			return err
		}
		return v.decls(n.Decls, id)
	case *ast.Package:
		return fmt.Errorf("GOTOTS_DEPRECATED_CONSTRUCT: ast.Package is not part of a parsed file tree")
	default:
		return fmt.Errorf("GOTOTS_UNVISITED_CONSTRUCT: no child-visit arm for %T", n)
	}
}

// The optional* helpers descend only into a present child; a nil concrete
// pointer is a genuinely absent edge, never a node.
func (v *visitor) optional(c *ast.CommentGroup, parent OccurrenceID) error {
	if c == nil {
		return nil
	}
	return v.node(c, parent)
}

func (v *visitor) optionalIdent(ident *ast.Ident, parent OccurrenceID) error {
	if ident == nil {
		return nil
	}
	return v.node(ident, parent)
}

func (v *visitor) optionalBasicLit(lit *ast.BasicLit, parent OccurrenceID) error {
	if lit == nil {
		return nil
	}
	return v.node(lit, parent)
}

func (v *visitor) optionalFieldList(list *ast.FieldList, parent OccurrenceID) error {
	if list == nil {
		return nil
	}
	return v.node(list, parent)
}

func (v *visitor) optionalExpr(x ast.Expr, parent OccurrenceID) error {
	if x == nil {
		return nil
	}
	return v.node(x, parent)
}

func (v *visitor) optionalStmt(s ast.Stmt, parent OccurrenceID) error {
	if s == nil {
		return nil
	}
	return v.node(s, parent)
}
