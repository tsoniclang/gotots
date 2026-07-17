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

	"github.com/tsoniclang/gotots/internal/goid"
	"github.com/tsoniclang/gotots/internal/typeid"
)

// qualifier renders package identity as the canonical import path.
func qualifier(p *types.Package) string { return p.Path() }

func typeString(t types.Type) (string, error) { return typeid.Canonical(t) }

// collectDeclarations walks one file's top-level declarations, producing
// both the identity records and the exact typed shapes. Every declared
// object must resolve through go/types; a missing definition is an error.
func collectDeclarations(p *packages.Package, file *ast.File, relativePath, scopeName, owner string, source []byte, stats *fileStats) error {
	info := p.TypesInfo
	fset := p.Fset
	pkgPath := p.PkgPath
	lineOf := func(pos token.Pos) int { return fset.Position(pos).Line }
	colOf := func(pos token.Pos) int { return fset.Position(pos).Column }

	// Every function literal in production scope is an independent
	// implementation unit: canonical position-qualified identity inside
	// its parent declaration, with the exact body hash.
	var funcLitErr error
	collectFuncLits := func(parentID string, node ast.Node) {
		if scopeName != "production" {
			return
		}
		ast.Inspect(node, func(n ast.Node) bool {
			lit, ok := n.(*ast.FuncLit)
			if !ok {
				return true
			}
			position := fset.Position(lit.Pos())
			id := goid.Repeatable(pkgPath, "funclit", "", relativePath, position.Line, position.Column)
			// Whole-literal span (func keyword through closing brace): the
			// signature is part of the identity so a parameter or result
			// type change is detected, matching the translator's span.
			start := fset.Position(lit.Pos()).Offset
			end := fset.Position(lit.End()).Offset
			if start < 0 || end > len(source) || start >= end {
				// A function literal always spans a non-empty, in-bounds
				// source range; an invalid span is a source/fileset mismatch
				// (a bug), never silently absent evidence.
				if funcLitErr == nil {
					funcLitErr = fmt.Errorf("funclit %s: invalid source span [%d,%d) over %d bytes", id, start, end, len(source))
				}
				return false
			}
			digest := sha256.Sum256(source[start:end])
			bodyHash := hex.EncodeToString(digest[:])
			stats.funcLitShapes = append(stats.funcLitShapes, FuncLitShape{ID: id, Parent: parentID, BodyHash: bodyHash})
			return true
		})
	}

	declare := func(kind, name, receiver string, node ast.Node, namePos token.Pos, body *ast.BlockStmt) (string, error) {
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
		if owner != pkgPath {
			declaration.Owner = owner
		}
		// Canonical identity derives from package and object identity, so
		// file moves inside a package never change it. Only Go's legally
		// repeatable declarations (package-level func init, blank
		// identifiers) carry file/position qualification — by the declared
		// identifier's own position, so `var _, _ = f()` stays unique.
		switch {
		case goid.IsRepeatable(kind, name):
			declaration.ID = goid.Repeatable(pkgPath, kind, name, relativePath, lineOf(namePos), colOf(namePos))
		case kind == "method":
			declaration.ID = goid.Method(pkgPath, receiver, name)
		case kind == "func":
			declaration.ID = goid.Func(pkgPath, name)
		default:
			declaration.ID = goid.Value(pkgPath, kind, name)
		}
		if body != nil {
			declaration.HasBody = true
			declaration.Statements = countStatements(body)
			start := fset.Position(body.Pos()).Offset
			end := fset.Position(body.End()).Offset
			if start < 0 || end > len(source) || start >= end {
				return "", fmt.Errorf("declaration %s has an invalid body span [%d, %d) in a %d-byte file", declaration.ID, start, end, len(source))
			}
			digest := sha256.Sum256(source[start:end])
			declaration.BodySha256 = hex.EncodeToString(digest[:])
		}
		stats.declarations = append(stats.declarations, declaration)
		return declaration.ID, nil
	}

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			kind := "func"
			receiver := ""
			if d.Recv != nil {
				kind = "method"
				receiver = receiverBaseName(d.Recv)
			}
			id, err := declare(kind, d.Name.Name, receiver, d, d.Name.Pos(), d.Body)
			if err != nil {
				return err
			}
			if err := shapeFunction(info, d, id, stats); err != nil {
				return err
			}
			if d.Body != nil {
				collectFuncLits(id, d.Body)
			}
			// The go test discovery contract applies only to functions
			// declared in _test.go files (the toolchain's documented
			// test-file rule), never to test-support production files.
			if scopeName == "test" && d.Recv == nil && strings.HasSuffix(relativePath, "_test.go") {
				recordTestFunction(info, d, id, relativePath, lineOf(d.Pos()), stats)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					kind := "var"
					if d.Tok == token.CONST {
						kind = "const"
					}
					for i, name := range s.Names {
						id, err := declare(kind, name.Name, "", s, name.Pos(), nil)
						if err != nil {
							return err
						}
						initializerHash := ""
						if kind == "var" && i < len(s.Values) {
							// The initializer's exact source bytes are
							// identity-bearing evidence. A PRESENT initializer
							// always spans a non-empty in-bounds range; an
							// invalid span is a source/fileset mismatch and a
							// hard error, never silently absent evidence.
							start := fset.Position(s.Values[i].Pos()).Offset
							end := fset.Position(s.Values[i].End()).Offset
							if start < 0 || end > len(source) || start >= end {
								return fmt.Errorf("var %s: invalid initializer span [%d,%d) over %d bytes", id, start, end, len(source))
							}
							digest := sha256.Sum256(source[start:end])
							initializerHash = hex.EncodeToString(digest[:])
						}
						if err := shapeValue(info, name, kind, id, initializerHash, stats); err != nil {
							return err
						}
						if kind == "var" && i < len(s.Values) {
							collectFuncLits(id, s.Values[i])
						}
					}
				case *ast.TypeSpec:
					kind := "type"
					if s.Assign.IsValid() {
						kind = "alias"
					}
					id, err := declare(kind, s.Name.Name, "", s, s.Name.Pos(), nil)
					if err != nil {
						return err
					}
					if err := shapeType(info, s, kind, id, stats); err != nil {
						return err
					}
				}
			}
		}
	}
	if funcLitErr != nil {
		return funcLitErr
	}
	return nil
}

func shapeFunction(info *types.Info, d *ast.FuncDecl, id string, stats *fileStats) error {
	object, ok := info.Defs[d.Name].(*types.Func)
	if !ok {
		return fmt.Errorf("declaration %s has no typed definition", id)
	}
	signature := object.Type().(*types.Signature)
	var terr error
	ts := tsFn(&terr)
	shape := FunctionShape{
		ID:        id,
		Variadic:  signature.Variadic(),
		Signature: ts(signature),
	}
	if recv := signature.Recv(); recv != nil {
		shape.Receiver = ts(recv.Type())
	}
	shape.TypeParams = typeParamShapes(signature.TypeParams(), &terr)
	if shape.TypeParams == nil {
		shape.TypeParams = typeParamShapes(signature.RecvTypeParams(), &terr)
	}
	shape.Params = tupleParams(signature.Params(), &terr)
	shape.Results = tupleParams(signature.Results(), &terr)
	if terr != nil {
		return terr
	}
	stats.functionShapes = append(stats.functionShapes, shape)
	return nil
}

func shapeValue(info *types.Info, name *ast.Ident, kind, id, initializerHash string, stats *fileStats) error {
	object := info.Defs[name]
	if object == nil {
		return fmt.Errorf("declaration %s has no typed definition", id)
	}
	var terr error
	ts := tsFn(&terr)
	if kind == "const" {
		constant, ok := object.(*types.Const)
		if !ok {
			return fmt.Errorf("declaration %s is not a constant", id)
		}
		shape := ConstShape{ID: id, Type: ts(constant.Type()), Value: constant.Val().ExactString()}
		if terr != nil {
			return terr
		}
		stats.constShapes = append(stats.constShapes, shape)
		return nil
	}
	shape := VarShape{ID: id, Type: ts(object.Type()), InitializerHash: initializerHash}
	if terr != nil {
		return terr
	}
	stats.varShapes = append(stats.varShapes, shape)
	return nil
}

func shapeType(info *types.Info, s *ast.TypeSpec, kind, id string, stats *fileStats) error {
	object, ok := info.Defs[s.Name].(*types.TypeName)
	if !ok {
		return fmt.Errorf("declaration %s has no typed definition", id)
	}
	var terr error
	ts := tsFn(&terr)
	if kind == "alias" {
		shape := AliasShape{ID: id, Target: ts(types.Unalias(object.Type()))}
		if terr != nil {
			return terr
		}
		stats.aliasShapes = append(stats.aliasShapes, shape)
		return nil
	}
	named, ok := object.Type().(*types.Named)
	if !ok {
		return fmt.Errorf("declaration %s is not a named type (%T)", id, object.Type())
	}
	kindName, err := namedKind(named)
	if err != nil {
		return fmt.Errorf("declaration %s: %w", id, err)
	}
	shape := TypeShape{
		ID:         id,
		Kind:       kindName,
		Underlying: ts(named.Underlying()),
		TypeParams: typeParamShapes(named.TypeParams(), &terr),
	}
	switch underlying := named.Underlying().(type) {
	case *types.Struct:
		for i := range underlying.NumFields() {
			field := underlying.Field(i)
			shape.Fields = append(shape.Fields, FieldShape{
				Name:     field.Name(),
				Type:     ts(field.Type()),
				Tag:      underlying.Tag(i),
				Embedded: field.Embedded(),
				Exported: field.Exported(),
			})
		}
	case *types.Interface:
		for i := range underlying.NumExplicitMethods() {
			method := underlying.ExplicitMethod(i)
			shape.InterfaceMethods = append(shape.InterfaceMethods, MethodShape{
				Name:      method.Name(),
				Signature: ts(method.Type()),
			})
		}
		for i := range underlying.NumEmbeddeds() {
			shape.InterfaceEmbeds = append(shape.InterfaceEmbeds, ts(underlying.EmbeddedType(i)))
		}
	}
	for i := range named.NumMethods() {
		method := named.Method(i)
		signature := method.Type().(*types.Signature)
		pointerReceiver := false
		if recv := signature.Recv(); recv != nil {
			_, pointerReceiver = recv.Type().(*types.Pointer)
		}
		shape.Methods = append(shape.Methods, MethodShape{
			Name:            method.Name(),
			Signature:       ts(signature),
			PointerReceiver: pointerReceiver,
		})
	}
	if terr != nil {
		return terr
	}
	stats.typeShapes = append(stats.typeShapes, shape)
	return nil
}

// tsFn returns a canonical-identity renderer that records the FIRST
// error into acc, so a shape builder can compose identities inline and
// then fail closed — the poison-string dual contract is eliminated.
func tsFn(acc *error) func(types.Type) string {
	return func(t types.Type) string {
		s, err := typeString(t)
		if err != nil && *acc == nil {
			*acc = err
		}
		return s
	}
}

func typeParamShapes(list *types.TypeParamList, acc *error) []TypeParamShape {
	if list == nil || list.Len() == 0 {
		return nil
	}
	ts := tsFn(acc)
	shapes := make([]TypeParamShape, 0, list.Len())
	for i := range list.Len() {
		parameter := list.At(i)
		shapes = append(shapes, TypeParamShape{
			Name:       parameter.Obj().Name(),
			Constraint: ts(parameter.Constraint()),
		})
	}
	return shapes
}

func tupleParams(tuple *types.Tuple, acc *error) []Param {
	if tuple == nil || tuple.Len() == 0 {
		return nil
	}
	ts := tsFn(acc)
	params := make([]Param, 0, tuple.Len())
	for i := range tuple.Len() {
		variable := tuple.At(i)
		params = append(params, Param{Name: variable.Name(), Type: ts(variable.Type())})
	}
	return params
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
