package census

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// fileStats collects the typed inventory of one source file.
type fileStats struct {
	declarations   []DeclarationRecord
	directives     []DirectiveRecord
	rare           []RareConstructRecord
	functionShapes []FunctionShape
	typeShapes     []TypeShape
	constShapes    []ConstShape
	varShapes      []VarShape
	aliasShapes    []AliasShape
	testFunctions   []TestFunctionRecord
	externalUses    map[string]*ExternalObligation
	externalObjects map[types.Object]bool
	constructs     map[string]int
	builtins       map[string]int
	rangeOperands  map[string]int
	indexOperands  map[string]int
	astKinds       map[string]int
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

// inspectFile walks one file with full type information. Construct
// classification uses go/types identity, never source spelling: a local
// function named append is not the builtin, and an imported package named
// maps is not the language map operation.
//
// pkgPath in records is the semantic package identity (p.PkgPath — a
// black-box test file keeps its p_test identity); owner is the package
// whose profile scope owns the file. isExternal classifies an object's
// defining package under the project profile.
func inspectFile(p *packages.Package, file *ast.File, relativePath, scopeName, owner string, source []byte, isExternal func(string) bool) (*fileStats, error) {
	stats := &fileStats{
		externalUses:    map[string]*ExternalObligation{},
		externalObjects: map[types.Object]bool{},
		constructs:      map[string]int{},
		builtins:        map[string]int{},
		rangeOperands:   map[string]int{},
		indexOperands:   map[string]int{},
		astKinds:        map[string]int{},
	}
	info := p.TypesInfo
	fset := p.Fset

	lineOf := func(pos token.Pos) int { return fset.Position(pos).Line }
	colOf := func(pos token.Pos) int { return fset.Position(pos).Column }
	rareOnly := func(construct string, pos token.Pos) {
		if rareConstructs[construct] {
			stats.rare = append(stats.rare, RareConstructRecord{
				Construct: construct, File: relativePath, Line: lineOf(pos), Col: colOf(pos),
			})
		}
	}
	record := func(construct string, pos token.Pos) {
		stats.constructs[construct]++
		rareOnly(construct, pos)
	}

	if err := collectDeclarations(p, file, relativePath, scopeName, owner, source, stats); err != nil {
		return nil, err
	}

	// Selector expressions in call position are classified by
	// classifyCall; the pre-pass marks them so standalone selections
	// (field reads, bound method values) are counted distinctly.
	calleeSelectors := map[*ast.SelectorExpr]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok {
				calleeSelectors[selector] = true
			}
		}
		return true
	})

	var inspectError error
	fail := func(err error) bool {
		inspectError = err
		return false
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil || inspectError != nil {
			return false
		}
		kind, err := astKindName(node)
		if err != nil {
			return fail(err)
		}
		stats.astKinds[kind]++

		switch n := node.(type) {
		case *ast.RangeStmt:
			record("range", n.Pos())
			class, err := typeClass(info.TypeOf(n.X))
			if err != nil {
				return fail(fmt.Errorf("range operand at %s:%d: %w", relativePath, lineOf(n.Pos()), err))
			}
			stats.rangeOperands[class]++
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
		case *ast.AssignStmt:
			stats.constructs["assign:"+n.Tok.String()]++
		case *ast.IncDecStmt:
			stats.constructs["incDec:"+n.Tok.String()]++
		case *ast.BinaryExpr:
			stats.constructs["binaryOp:"+n.Op.String()]++
		case *ast.BranchStmt:
			switch n.Tok {
			case token.GOTO:
				record("goto", n.Pos())
			case token.FALLTHROUGH:
				record("fallthrough", n.Pos())
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
				record("channelReceive", n.Pos())
			case token.AND:
				record("addressOf", n.Pos())
			default:
				stats.constructs["unaryOp:"+n.Op.String()]++
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
			if err := recordExternalUse(info, n, isExternal, stats); err != nil {
				return fail(fmt.Errorf("%s:%d: %w", relativePath, lineOf(n.Pos()), err))
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
			class, err := typeClass(info.TypeOf(n.X))
			if err != nil {
				return fail(fmt.Errorf("index operand at %s:%d: %w", relativePath, lineOf(n.Pos()), err))
			}
			stats.indexOperands[class]++
		case *ast.IndexListExpr:
			// Always an instantiation; counted at its Ident.
		case *ast.CallExpr:
			if err := classifyCall(info, n, stats, rareOnly); err != nil {
				return fail(fmt.Errorf("%s: %w", relativePath, err))
			}
		case *ast.SelectorExpr:
			if selection, ok := info.Selections[n]; ok && !calleeSelectors[n] {
				switch selection.Kind() {
				case types.FieldVal:
					stats.constructs["fieldAccess"]++
				case types.MethodVal:
					record("methodValue", n.Pos())
				case types.MethodExpr:
					record("methodExpr", n.Pos())
				}
			}
			if obj := info.Uses[n.Sel]; obj != nil && obj.Pkg() == types.Unsafe {
				record("unsafe", n.Pos())
			}
		}
		return true
	})
	if inspectError != nil {
		return nil, fmt.Errorf("%s: %w", relativePath, inspectError)
	}

	collectDirectives(file, relativePath, lineOf, colOf, stats)
	return stats, nil
}

// recordExternalUse records one reference to an object defined in an
// external package: a deterministic declaration/stub obligation. Object
// classes outside the reviewed set fail closed.
func recordExternalUse(info *types.Info, ident *ast.Ident, isExternal func(string) bool, stats *fileStats) error {
	object := info.Uses[ident]
	if object == nil || object.Pkg() == nil || !isExternal(object.Pkg().Path()) {
		return nil
	}
	// Objects of the compiler-recognized unsafe package are language-level
	// operations, not library obligations: every occurrence is already
	// pinned as an "unsafe" rare-construct record, and its disposition is
	// the unsupported/manual language-operation policy, never a stub.
	if object.Pkg() == types.Unsafe {
		return nil
	}
	name := object.Name()
	var kind string
	switch concrete := object.(type) {
	case *types.Func:
		kind = "func"
		if qualified, isMethod := methodQualifiedName(concrete); isMethod {
			kind = "method"
			name = qualified
		}
	case *types.Var:
		if concrete.IsField() {
			kind = "field"
		} else {
			kind = "var"
		}
	case *types.Const:
		kind = "const"
	case *types.TypeName:
		kind = "type"
	case *types.PkgName:
		return nil // the import itself, not an object obligation
	default:
		return fmt.Errorf("unreviewed external object class %T for %s.%s", object, object.Pkg().Path(), object.Name())
	}
	key := object.Pkg().Path() + "\x00" + name + "\x00" + kind
	obligation := stats.externalUses[key]
	if obligation == nil {
		obligation = &ExternalObligation{Package: object.Pkg().Path(), Name: name, Kind: kind}
		stats.externalUses[key] = obligation
	}
	obligation.Uses++
	stats.externalObjects[object] = true
	return nil
}

// methodQualifiedName returns "Receiver.Method" for methods, so methods of
// different types with the same name keep distinct identities.
func methodQualifiedName(function *types.Func) (string, bool) {
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return "", false
	}
	recv := signature.Recv().Type()
	if pointer, ok := recv.(*types.Pointer); ok {
		recv = pointer.Elem()
	}
	if named, ok := types.Unalias(recv).(*types.Named); ok {
		return named.Obj().Name() + "." + function.Name(), true
	}
	if _, ok := recv.Underlying().(*types.Interface); ok {
		return "", false // interface methods keep their own name
	}
	return "", false
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

// knownDirectives is the reviewed set of compiler directives. An
// occurrence outside this set is recorded with Known=false and blocks an
// authoritative result until it receives a disposition.
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

// collectDirectives records every compiler directive occurrence. The
// toolchain contract for directive comments is a line comment whose text
// begins exactly "//go:" with no space; //line and /*line*/ position
// directives and cgo //export markers are also recognized.
func collectDirectives(file *ast.File, relativePath string, lineOf, colOf func(token.Pos) int, stats *fileStats) {
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
				Col:       colOf(comment.Pos()),
				Directive: name,
				Known:     knownDirectives[name],
			})
		}
	}
}
