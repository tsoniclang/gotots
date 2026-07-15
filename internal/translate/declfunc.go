// Function-declaration translation and module-environment helpers.
package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/goid"
	"github.com/tsoniclang/gotots/internal/ir"
)

func translateFunc(p *packages.Package, sourceDir string, unit ir.Scope, relativeFile string, source []byte, decl *ast.FuncDecl, options Options) (*ir.Func, *Proof, error) {
	name := decl.Name.Name
	id := goid.Func(p.PkgPath, name)
	if decl.Recv != nil {
		id = goid.Method(p.PkgPath, receiverBase(decl.Recv), name)
	}

	bodyHash := ""
	if decl.Body != nil {
		start := p.Fset.Position(decl.Body.Pos()).Offset
		end := p.Fset.Position(decl.Body.End()).Offset
		if start < 0 || end > len(source) || start >= end {
			return nil, nil, fmt.Errorf("declaration %s has an invalid body span", id)
		}
		digest := sha256.Sum256(source[start:end])
		bodyHash = hex.EncodeToString(digest[:])
	}

	function, err := ir.BuildFunc(p, sourceDir, unit, decl, id, bodyHash)
	if err != nil {
		return nil, nil, err
	}

	object := p.TypesInfo.Defs[decl.Name].(*types.Func)
	signatureText := types.TypeString(object.Type(), func(pkg *types.Package) string { return pkg.Path() })
	signatureDigest := sha256.Sum256([]byte(signatureText))

	representations := map[string]string{}
	recordRepresentation := func(t ir.Type) {
		representations[t.Go] = conservativeCarrier(t)
	}
	if function.Receiver != nil {
		recordRepresentation(function.Receiver.Type)
	}
	for _, parameter := range function.Params {
		recordRepresentation(parameter.Type)
	}
	for _, result := range function.Results {
		recordRepresentation(result.Type)
	}

	return function, &Proof{
		ID:              id,
		SourceRevision:  options.SourceRevision,
		Package:         p.PkgPath,
		File:            relativeFile,
		SignatureHash:   hex.EncodeToString(signatureDigest[:]),
		BodyHash:        bodyHash,
		Operations:      function.Operations,
		Representations: representations,
		LoweringPlan:    LoweringPlanV1,
		GeneratedSymbol: name,
	}, nil
}
