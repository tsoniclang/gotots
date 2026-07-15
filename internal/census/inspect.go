package census

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// fileStats collects the typed inventory of one source file.
type fileStats struct {
	declarations  []DeclarationRecord
	directives    []DirectiveRecord
	rare          []RareConstructRecord
	constructs    map[string]int
	builtins      map[string]int
	rangeOperands map[string]int
	indexOperands map[string]int
	astKinds      map[string]int
}

// rareConstructs are low-volume constructs whose every occurrence is
// recorded with an exact location.
var rareConstructs = map[string]bool{
	"go":             true,
	"select":         true,
	"channelSend":    true,
	"channelReceive": true,
	"goto":           true,
	"fallthrough":    true,
	"fullSliceExpr":  true,
	"unsafe":         true,
	"recover":        true,
}

// knownDirectives is the reviewed set of compiler directives. An occurrence
// outside this set is recorded with Known=false and blocks generation until
// it receives a disposition.
var knownDirectives = map[string]bool{
	"go:build":            true,
	"go:embed":            true,
	"go:generate":         true,
	"go:linkname":         true,
	"go:noinline":         true,
	"go:nosplit":          true,
	"go:noescape":         true,
	"go:norace":           true,
	"go:nocheckptr":       true,
	"go:uintptrescapes":   true,
	"go:uintptrkeepalive": true,
	"line":                true,
	"export":              true,
}

// inspectFile walks one file with full type information. Construct
// classification uses go/types identity, never source spelling: a local
// function named append is not the builtin, and an imported package named
// maps is not the language map operation.
func inspectFile(p *packages.Package, file *ast.File, relativePath, scopeName string, source []byte) (*fileStats, error) {
	stats := &fileStats{
		constructs:    map[string]int{},
		builtins:      map[string]int{},
		rangeOperands: map[string]int{},
		indexOperands: map[string]int{},
		astKinds:      map[string]int{},
	}
	info := p.TypesInfo
	fset := p.Fset
	pkgPath := classificationPath(p)

	lineOf := func(pos token.Pos) int { return fset.Position(pos).Line }
	rareOnly := func(construct string, pos token.Pos) {
		if rareConstructs[construct] {
			stats.rare = append(stats.rare, RareConstructRecord{
				Construct: construct, File: relativePath, Line: lineOf(pos),
			})
		}
	}
	record := func(construct string, pos token.Pos) {
		stats.constructs[construct]++
		rareOnly(construct, pos)
	}

	declare := func(kind, name, receiver string, node ast.Node, body *ast.BlockStmt) {
		declaration := DeclarationRecord{
			Package:   pkgPath,
			File:      relativePath,
			Kind:      kind,
			Name:      name,
			Receiver:  receiver,
			Exported:  token.IsExported(name),
			Scope:     scopeName,
			StartLine: lineOf(node.Pos()),
			EndLine:   lineOf(node.End()),
		}
		qualified := name
		if receiver != "" {
			qualified = receiver + "." + name
		}
		declaration.ID = pkgPath + "::" + relativePath + "::" + kind + "::" + qualified
		if body != nil {
			declaration.HasBody = true
			declaration.Statements = countStatements(body)
			start := fset.Position(body.Pos()).Offset
			end := fset.Position(body.End()).Offset
			if start < 0 || end > len(source) || start > end {
				start, end = 0, 0
			}
			digest := sha256.Sum256(source[start:end])
			declaration.BodySha256 = hex.EncodeToString(digest[:])
		}
		stats.declarations = append(stats.declarations, declaration)
	}

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv != nil {
				declare("method", d.Name.Name, receiverBaseName(d.Recv), d, d.Body)
			} else {
				declare("func", d.Name.Name, "", d, d.Body)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					kind := "var"
					if d.Tok == token.CONST {
						kind = "const"
					}
					for _, name := range s.Names {
						declare(kind, name.Name, "", s, nil)
					}
				case *ast.TypeSpec:
					if s.Assign.IsValid() {
						declare("alias", s.Name.Name, "", s, nil)
					} else {
						declare("type", s.Name.Name, "", s, nil)
					}
				}
			}
		}
	}

	var inspectError error
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil || inspectError != nil {
			return false
		}
		kind, err := astKindName(node)
		if err != nil {
			inspectError = err
			return false
		}
		stats.astKinds[kind]++

		switch n := node.(type) {
		case *ast.RangeStmt:
			record("range", n.Pos())
			stats.rangeOperands[typeClass(info.TypeOf(n.X))]++
		case *ast.DeferStmt:
			record("defer", n.Pos())
		case *ast.GoStmt:
			record("go", n.Pos())
		case *ast.SelectStmt:
			record("select", n.Pos())
		case *ast.TypeSwitchStmt:
			record("typeSwitch", n.Pos())
		case *ast.SwitchStmt:
			record("switch", n.Pos())
		case *ast.SendStmt:
			record("channelSend", n.Pos())
		case *ast.LabeledStmt:
			record("label", n.Pos())
		case *ast.BranchStmt:
			switch n.Tok {
			case token.GOTO:
				record("goto", n.Pos())
			case token.FALLTHROUGH:
				record("fallthrough", n.Pos())
			case token.BREAK:
				if n.Label != nil {
					record("labeledBreak", n.Pos())
				}
			case token.CONTINUE:
				if n.Label != nil {
					record("labeledContinue", n.Pos())
				}
			}
		case *ast.UnaryExpr:
			switch n.Op {
			case token.ARROW:
				record("channelReceive", n.Pos())
			case token.AND:
				record("addressOf", n.Pos())
			}
		case *ast.StarExpr:
			if tv, ok := info.Types[n.X]; ok && tv.IsType() {
				record("pointerType", n.Pos())
			} else {
				record("dereference", n.Pos())
			}
		case *ast.SliceExpr:
			if n.Slice3 {
				record("fullSliceExpr", n.Pos())
			} else {
				record("reslice", n.Pos())
			}
		case *ast.TypeAssertExpr:
			if n.Type != nil { // Type==nil is the type-switch header form.
				record("typeAssert", n.Pos())
			}
		case *ast.FuncLit:
			record("funcLit", n.Pos())
		case *ast.Ident:
			// Instances covers both explicit (F[int]) and inferred generic
			// instantiations of functions and types.
			if _, ok := info.Instances[n]; ok {
				record("genericInstantiation", n.Pos())
			}
		case *ast.IndexExpr:
			// A value-level generic instantiation F[int] is an IndexExpr
			// whose operand identifier appears in Instances; it was counted
			// there and is not an index operation.
			if isInstantiation(info, n.X) {
				break
			}
			if tv, ok := info.Types[n]; ok && tv.IsType() {
				break // type instantiation, counted at its Ident
			}
			stats.indexOperands[typeClass(info.TypeOf(n.X))]++
		case *ast.IndexListExpr:
			// Always an instantiation; counted at its Ident.
		case *ast.CallExpr:
			classifyCall(info, n, stats, rareOnly)
		case *ast.SelectorExpr:
			if obj := info.Uses[n.Sel]; obj != nil && obj.Pkg() == types.Unsafe {
				record("unsafe", n.Pos())
			}
		}
		return true
	})
	if inspectError != nil {
		return nil, fmt.Errorf("%s: %w", relativePath, inspectError)
	}

	// Directive census. The toolchain contract for directive comments is a
	// line comment whose text begins exactly "//go:" with no space; //line
	// and /*line*/ position directives and cgo //export markers are also
	// recognized.
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := comment.Text
			var name string
			if rest, ok := strings.CutPrefix(text, "//go:"); ok {
				directive, _, _ := strings.Cut(rest, " ")
				name = "go:" + directive
			} else if strings.HasPrefix(text, "//export ") {
				name = "export"
			} else if strings.HasPrefix(text, "//line ") || strings.HasPrefix(text, "/*line ") {
				name = "line"
			} else {
				continue
			}
			stats.directives = append(stats.directives, DirectiveRecord{
				File:      relativePath,
				Line:      lineOf(comment.Pos()),
				Directive: name,
				Known:     knownDirectives[name],
			})
		}
	}
	return stats, nil
}

func isInstantiation(info *types.Info, fun ast.Expr) bool {
	switch f := ast.Unparen(fun).(type) {
	case *ast.Ident:
		_, ok := info.Instances[f]
		return ok
	case *ast.SelectorExpr:
		_, ok := info.Instances[f.Sel]
		return ok
	}
	return false
}

func receiverBaseName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	for {
		switch base := t.(type) {
		case *ast.StarExpr:
			t = base.X
		case *ast.IndexExpr:
			t = base.X
		case *ast.IndexListExpr:
			t = base.X
		case *ast.ParenExpr:
			t = base.X
		case *ast.Ident:
			return base.Name
		default:
			return ""
		}
	}
}

func countStatements(body *ast.BlockStmt) int {
	count := 0
	ast.Inspect(body, func(node ast.Node) bool {
		if _, ok := node.(ast.Stmt); ok {
			count++
		}
		return true
	})
	return count
}

func classifyCall(info *types.Info, call *ast.CallExpr, stats *fileStats, rareOnly func(string, token.Pos)) {
	// Conversion: the operand in call position is a type.
	if tv, ok := info.Types[call.Fun]; ok && tv.IsType() {
		stats.constructs["conversion"]++
		return
	}

	var ident *ast.Ident
	switch f := ast.Unparen(call.Fun).(type) {
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
		rareOnly(builtin.Name(), call.Pos())
	}
}

// typeClass names the semantic operand class of a type for range/index
// census purposes, resolving named types to their underlying form.
func typeClass(t types.Type) string {
	if t == nil {
		return "unknown"
	}
	if _, ok := types.Unalias(t).(*types.TypeParam); ok {
		return "typeParam"
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
		return "other"
	}
}

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
