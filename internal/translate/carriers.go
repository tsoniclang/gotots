// Carrier naming and source-position helpers shared across the
// translation passes.
package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/ir"
)

func conservativeCarrier(t ir.Type) string {
	switch {
	case t.Kind == ir.KindBool:
		return "boolean"
	case t.Kind == ir.KindString:
		return "js-string(equality-only-ordering-unsupported)"
	case t.Kind == ir.KindPointer:
		return "object-identity-nilable(undefined)"
	case t.Kind == ir.KindStruct:
		return "class-instance-value(copy-on-bind,in-place-store)"
	case t.Kind == ir.KindFunc:
		return "js-closure-nilable(undefined)-capture-by-reference"
	case t.Kind == ir.KindIface:
		return "iface-box-nilable(undefined)-rtti-identity"
	case t.Kind == ir.KindMap:
		return "js-map-nilable(undefined)-has-based-lookup"
	case t.Kind == ir.KindSlice:
		return "goslice-view-nilable(undefined)-conservative"
	case t.Kind.Float():
		return "number"
	case t.Kind.Wide64():
		return "bigint-exact-64"
	case t.Kind == ir.KindUnit:
		return "unit-literal-0"
	case t.Kind == ir.KindArray:
		return "native-array-of-element-carrier"
	case t.Kind == ir.KindExternal:
		return "external-branded-handle"
	case t.Kind.Integer():
		return fmt.Sprintf("number-wrapped-%d", t.Kind.Bits())
	case t.TypeParamName != "":
		return "type-parameter-instantiation"
	}
	// Every reviewed kind is named above; an unnamed kind is a
	// contradiction the representation gate rejects.
	return "unreviewed-kind(" + t.Go + ")"
}

func spanOf(p *packages.Package, sourceDir string, pos token.Pos) ir.Span {
	position := p.Fset.Position(pos)
	file := position.Filename
	if relative, err := filepath.Rel(sourceDir, file); err == nil && !strings.HasPrefix(relative, "..") {
		file = filepath.ToSlash(relative)
	}
	return ir.Span{File: file, Line: position.Line, Col: position.Column}
}

// relativeImport computes the ESM specifier from one bundle directory to a
// bundle path, always with an explicit ./ or ../ prefix.
func relativeImport(fromDir, to string) (string, error) {
	relative, err := filepath.Rel(filepath.FromSlash(fromDir), filepath.FromSlash(to))
	if err != nil {
		return "", err
	}
	specifier := filepath.ToSlash(relative)
	if !strings.HasPrefix(specifier, ".") {
		specifier = "./" + specifier
	}
	return specifier, nil
}

// blankInitHash hashes a blank variable's initializer source exactly as
// the census records it.
func blankInitHash(p *packages.Package, source []byte, initExpr ast.Expr) (string, error) {
	if initExpr == nil {
		return "", nil
	}
	start := p.Fset.Position(initExpr.Pos()).Offset
	end := p.Fset.Position(initExpr.End()).Offset
	if start < 0 || end > len(source) || start >= end {
		// A present blank-var initializer always spans a valid range; an
		// invalid span is a hard error, never silently absent evidence.
		return "", fmt.Errorf("blank var initializer: invalid span [%d,%d) over %d bytes", start, end, len(source))
	}
	digest := sha256.Sum256(source[start:end])
	return hex.EncodeToString(digest[:]), nil
}
