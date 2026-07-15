package census

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"
)

// fileStats collects the typed inventory of one source file.
type fileStats struct {
	decls         DeclCounts
	bodies        int
	statements    int
	constructs    map[string]int
	builtins      map[string]int
	rangeOperands map[string]int
	indexOperands map[string]int
	directives    map[string]int
	astKinds      map[string]int
}

// inspectFile walks one file with full type information. Construct
// classification uses go/types identity, never source spelling: a local
// function named append is not the builtin, and an imported package named
// maps is not the language map operation.
func inspectFile(p *packages.Package, file *ast.File) *fileStats {
	stats := &fileStats{
		constructs:    map[string]int{},
		builtins:      map[string]int{},
		rangeOperands: map[string]int{},
		indexOperands: map[string]int{},
		directives:    map[string]int{},
		astKinds:      map[string]int{},
	}
	info := p.TypesInfo

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv != nil {
				stats.decls.Methods++
			} else {
				stats.decls.Functions++
			}
			if d.Body == nil {
				stats.decls.BodylessFunctions++
			} else {
				stats.bodies++
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					if d.Tok == token.CONST {
						stats.decls.Constants += len(s.Names)
					} else if d.Tok == token.VAR {
						stats.decls.Variables += len(s.Names)
					}
				case *ast.TypeSpec:
					if s.Assign.IsValid() {
						stats.decls.Aliases++
					} else {
						stats.decls.NamedTypes++
					}
				}
			}
		}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		kind := reflect.TypeOf(node).String()
		stats.astKinds[strings.TrimPrefix(kind, "*ast.")]++

		if _, ok := node.(ast.Stmt); ok {
			stats.statements++
		}

		switch n := node.(type) {
		case *ast.RangeStmt:
			stats.constructs["range"]++
			stats.rangeOperands[typeClass(info.TypeOf(n.X))]++
		case *ast.DeferStmt:
			stats.constructs["defer"]++
		case *ast.GoStmt:
			stats.constructs["go"]++
		case *ast.SelectStmt:
			stats.constructs["select"]++
		case *ast.TypeSwitchStmt:
			stats.constructs["typeSwitch"]++
		case *ast.SwitchStmt:
			stats.constructs["switch"]++
		case *ast.SendStmt:
			stats.constructs["channelSend"]++
		case *ast.LabeledStmt:
			stats.constructs["label"]++
		case *ast.BranchStmt:
			switch n.Tok {
			case token.GOTO:
				stats.constructs["goto"]++
			case token.FALLTHROUGH:
				stats.constructs["fallthrough"]++
			case token.BREAK:
				if n.Label != nil {
					stats.constructs["labeledBreak"]++
				}
			case token.CONTINUE:
				if n.Label != nil {
					stats.constructs["labeledContinue"]++
				}
			}
		case *ast.UnaryExpr:
			switch n.Op {
			case token.ARROW:
				stats.constructs["channelReceive"]++
			case token.AND:
				stats.constructs["addressOf"]++
			}
		case *ast.StarExpr:
			if tv, ok := info.Types[n.X]; ok && tv.IsType() {
				stats.constructs["pointerType"]++
			} else {
				stats.constructs["dereference"]++
			}
		case *ast.SliceExpr:
			if n.Slice3 {
				stats.constructs["fullSliceExpr"]++
			} else {
				stats.constructs["reslice"]++
			}
		case *ast.TypeAssertExpr:
			if n.Type != nil { // Type==nil is the type-switch header form.
				stats.constructs["typeAssert"]++
			}
		case *ast.FuncLit:
			stats.constructs["funcLit"]++
		case *ast.IndexExpr:
			if tv, ok := info.Types[n]; ok && tv.IsType() {
				stats.constructs["genericInstantiation"]++
			} else {
				stats.indexOperands[typeClass(info.TypeOf(n.X))]++
			}
		case *ast.IndexListExpr:
			stats.constructs["genericInstantiation"]++
		case *ast.CallExpr:
			classifyCall(info, n, stats)
		case *ast.SelectorExpr:
			if sel, ok := info.Selections[n]; ok {
				_ = sel
			} else if obj := info.Uses[n.Sel]; obj != nil && obj.Pkg() == types.Unsafe {
				stats.constructs["unsafe"]++
			}
		}
		return true
	})

	// Directive census: exact "//go:" prefixed comments per the toolchain
	// directive contract, plus cgo export markers.
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := comment.Text
			if rest, ok := strings.CutPrefix(text, "//go:"); ok {
				name, _, _ := strings.Cut(rest, " ")
				stats.directives["go:"+name]++
			} else if strings.HasPrefix(text, "//export ") {
				stats.directives["export"]++
			} else if strings.HasPrefix(text, "//line ") || strings.HasPrefix(text, "/*line ") {
				stats.directives["line"]++
			}
		}
	}
	return stats
}

func classifyCall(info *types.Info, call *ast.CallExpr, stats *fileStats) {
	fun := ast.Unparen(call.Fun)

	// Conversion: the operand in call position is a type.
	if tv, ok := info.Types[call.Fun]; ok && tv.IsType() {
		stats.constructs["conversion"]++
		return
	}

	var ident *ast.Ident
	switch f := fun.(type) {
	case *ast.Ident:
		ident = f
	case *ast.SelectorExpr:
		ident = f.Sel
	}
	if ident == nil {
		return
	}
	if builtin, ok := info.Uses[ident].(*types.Builtin); ok {
		stats.builtins[builtin.Name()]++
	}
}

// typeClass names the semantic operand class of a type for range/index
// census purposes, resolving named types to their underlying form.
func typeClass(t types.Type) string {
	if t == nil {
		return "unknown"
	}
	switch u := t.Underlying().(type) {
	case *types.Slice:
		return "slice"
	case *types.Array:
		return "array"
	case *types.Map:
		return "map"
	case *types.Chan:
		return "chan"
	case *types.Signature:
		return "func"
	case *types.Interface:
		if tp, ok := types.Unalias(t).(*types.TypeParam); ok {
			_ = tp
			return "typeParam"
		}
		return "interface"
	case *types.Pointer:
		if _, ok := u.Elem().Underlying().(*types.Array); ok {
			return "pointerToArray"
		}
		return "pointer"
	case *types.Basic:
		switch {
		case u.Info()&types.IsString != 0:
			return "string"
		case u.Info()&types.IsInteger != 0:
			return "integer"
		default:
			return "basic:" + u.Name()
		}
	default:
		if _, ok := types.Unalias(t).(*types.TypeParam); ok {
			return "typeParam"
		}
		return "other"
	}
}
