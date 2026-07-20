// Function and method translation for one unit package — pass 2 of
// translatePackage, split out by responsibility: functions and methods
// are attached to their receiver classes, or standalone for receivers
// erased to carriers; package init functions take synthesized names and
// run at module evaluation in file order, after every variable
// initializer — exactly Go.
package translate

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/facts"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/tsident"

	"golang.org/x/tools/go/packages"
)

// translateFunctionsPass translates every function and method of the
// package, appends body-support ledger entries, and reports how many
// bodies are unimplemented (materializing as typed placeholders).
func translateFunctionsPass(out *Generated, p *packages.Package, sourceDir string, unit ir.Scope, options Options, files []fileSource, structs map[string]*ir.Struct, corePath string, ledger []BodySupport) ([]*ir.Func, []emit.Method, []string, []BodySupport, int, error) {
	unimplementedUnits := 0
	var functions []*ir.Func
	var carrierMethods []emit.Method
	var initCalls []string
	blankFuncs := 0
	for _, f := range files {
		for _, decl := range f.file.Decls {
			funcDecl, isFunc := decl.(*ast.FuncDecl)
			if !isFunc {
				continue
			}
			function, proof, err := translateFunc(p, sourceDir, unit, f.relative, f.source, funcDecl, options)
			if err != nil {
				return nil, nil, nil, nil, 0, err
			}
			if funcDecl.Recv == nil && funcDecl.Name.Name == "init" {
				function.Name = fmt.Sprintf("init$%d", len(initCalls))
				// The proof records the exact emitted symbol.
				proof.GeneratedSymbol = tsident.EscapeDeclared(function.Name)
				if function.Support == ir.SupportIRAdmitted {
					initCalls = append(initCalls, function.Name)
				}
			}
			if funcDecl.Recv == nil && funcDecl.Name.Name == "_" {
				// A blank function is uncallable in Go; its body still
				// typechecks (stringer drift guards). It emits under a
				// package-unique name, exactly like init functions —
				// deterministic in file walk order.
				function.Name = fmt.Sprintf("_$%d", blankFuncs)
				blankFuncs++
				proof.GeneratedSymbol = function.Name
			}
			if funcDecl.Recv != nil {
				recvField := funcDecl.Recv.List[0]
				recvName := ""
				if len(recvField.Names) > 0 {
					recvName = recvField.Names[0].Name
				}
				_, pointerReceiver := recvField.Type.(*ast.StarExpr)
				fact := facts.AnalyzeReceiverNilability(recvName, pointerReceiver, funcDecl.Body)
				carrier := false
				if fnObj := p.TypesInfo.Defs[funcDecl.Name]; fnObj != nil {
					if signature, ok := fnObj.Type().(*types.Signature); ok && signature.Recv() != nil {
						recvType := signature.Recv().Type()
						if pointer, ok := recvType.(*types.Pointer); ok {
							recvType = pointer.Elem()
						}
						_, isStruct := recvType.Underlying().(*types.Struct)
						carrier = !isStruct
					}
				}
				planKey := function.PlanKey
				if planKey == "" {
					planKey = function.ID
				}
				out.NilabilityFacts = append(out.NilabilityFacts, NilabilityFact{
					ID:                planKey,
					CensusID:          function.ID,
					EquivalentAtEntry: fact.EquivalentAtEntry,
					ToleratesNil:      fact.ToleratesNil,
					GenericReceiver:   receiverIsGeneric(recvField.Type),
					CarrierReceiver:   carrier,
				})
			}
			ledger = append(ledger, BodySupport{
				ID: function.ID, Package: p.PkgPath, Kind: "body", State: function.Support, Sites: function.Sites,
			})
			if function.Support == ir.SupportIRAdmitted {
				registry, err := supportRegistry()
				if err != nil {
					return nil, nil, nil, nil, 0, err
				}
				for _, operation := range function.Operations {
					if !registry.Generated(operation) {
						return nil, nil, nil, nil, 0, fmt.Errorf("GOTOTS_CENSUS_UNREGISTERED_CLASS:\ngenerated body %s uses operation class %q with no reviewed support decision in contracts/support-classes.json",
							function.ID, operation)
					}
				}
			}
			if function.Support == ir.SupportUnimplemented {
				// Every unsupported site is on the record. The body still
				// MATERIALIZES as a typed throwing placeholder: its exact
				// signature typechecks so sibling and cross-package callers
				// compile, while calling it fails closed. The package is
				// publication-withheld below, but one unresolved body no
				// longer prevents its siblings from being emitted and checked.
				unimplementedUnits++
				function.Placeholder = true
			} else {
				proof.GeneratedFile = corePath
				out.Proofs = append(out.Proofs, *proof)
			}
			if function.Receiver == nil {
				functions = append(functions, function)
				if fnObj, isFunc := p.TypesInfo.Defs[funcDecl.Name].(*types.Func); isFunc && ir.HasEncFamilyFuncInstances(unit, fnObj) {
					encFn := *function
					encFn.FamilyEnc = true
					functions = append(functions, &encFn)
				}
				if fnObj, isFunc := p.TypesInfo.Defs[funcDecl.Name].(*types.Func); isFunc && ir.HasPtrCellFuncInstances(unit, fnObj) {
					ptrFn := *function
					ptrFn.FamilyPtrCell = true
					functions = append(functions, &ptrFn)
				}
				continue
			}
			owner := receiverBase(funcDecl.Recv)
			// An ALIAS receiver spelling (type ImportAttributesNode = Node)
			// must emit under the CANONICAL named type: every dispatch and
			// call site resolves through go/types and spells Node$Method,
			// so the declared function name must coincide.
			if fnObj, ok := p.TypesInfo.Defs[funcDecl.Name].(*types.Func); ok {
				recvType := fnObj.Type().(*types.Signature).Recv().Type()
				if pointer, isPointer := recvType.(*types.Pointer); isPointer {
					recvType = pointer.Elem()
				}
				if named, isNamed := types.Unalias(recvType).(*types.Named); isNamed {
					owner = named.Obj().Name()
				}
			}
			if owner == "" {
				return nil, nil, nil, nil, 0, fmt.Errorf("method %s has no named receiver type in package %s", function.Name, p.PkgPath)
			}
			if structDecl, ok := structs[owner]; ok {
				structDecl.Methods = append(structDecl.Methods, function)
				continue
			}
			carrierMethods = append(carrierMethods, emit.Method{TypeName: owner, Fn: function})
		}
	}
	return functions, carrierMethods, initCalls, ledger, unimplementedUnits, nil
}

// receiverIsGeneric reports whether the receiver type expression names
// a generic type (T[K] / T[K, V]).
func receiverIsGeneric(expr ast.Expr) bool {
	for {
		switch e := expr.(type) {
		case *ast.StarExpr:
			expr = e.X
		case *ast.ParenExpr:
			expr = e.X
		case *ast.IndexExpr, *ast.IndexListExpr:
			return true
		default:
			return false
		}
	}
}
