package source

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/identity"
)

// censusUniverse extracts the total top-level implementation-unit census of
// every file, computes overlay-aware source-span hashes and per-unit cgo
// evidence, parses cgo originals transiently, and joins checked-view
// declarations to their origins through //line evidence.
func censusUniverse(u *Universe) error {
	for _, pkg := range u.packages {
		for _, file := range pkg.files {
			if err := censusFile(u, pkg, file); err != nil {
				return err
			}
		}
		if err := resolveCgo(u, pkg); err != nil {
			return err
		}
		if pkg.disposition == DispositionOrdinarySource {
			// Every ordinary package carries its implicit initialization unit
			// in the pre-scope ledger with a typed catalog identity.
			implicit, err := identity.NewImplicitUnitID(pkg.id, identity.ImplicitOpPackageInit)
			if err != nil {
				return err
			}
			pkg.implicitUnits = append(pkg.implicitUnits, implicit)
		}
	}
	return nil
}

// censusFile censuses one file. Cgo originals (no checked syntax at their own
// path) are parsed transiently for the census; the transient tree is dropped
// before return.
func censusFile(u *Universe, pkg *LoadedPackage, file *LoadedFile) error {
	if raw, err := fileBytes(file.path, u.request.Overlay); err == nil {
		file.byteDigest = sha256.Sum256(raw)
	}
	if pkg.disposition == DispositionBuiltinUniverse || pkg.disposition == DispositionUnsafeIntrinsic {
		return nil // intrinsic contracts carry no censused source units
	}
	syntax := file.syntax
	fset := file.fset
	if syntax == nil && file.cgoOriginal {
		transientFset := token.NewFileSet()
		parsed, err := parseFileBytes(transientFset, file.path, u.request.Overlay)
		if err != nil {
			return &LoadError{Dir: u.request.Dir, Reason: file.id.String() + ": cgo original unparsable: " + err.Error()}
		}
		syntax = parsed
		fset = transientFset
	}
	if syntax == nil {
		return nil
	}
	raw, err := fileBytes(file.path, u.request.Overlay)
	if err != nil {
		return &LoadError{Dir: u.request.Dir, Reason: file.id.String() + ": source bytes unreadable: " + err.Error()}
	}
	// Per-unit cgo C-dependence is NOT decided here: it is derived after the
	// census from authoritative checked/type evidence (resolveCgo), never from
	// the "C" spelling or a hand-maintained scope model. The census only
	// establishes unit identity, span, and hash.
	//
	// The census mode is contract-derived acquisition data resolved before
	// the load: recursive-mode files census interiors locally; manifest-mode
	// files use a bounded top-level census and join their interior units from
	// the request's verified content-addressed manifest below. Either way the
	// pre-scope ledger is total, including every nested function literal.
	recursive := file.censusMode == CensusRecursive
	for _, decl := range syntax.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			kind := identity.UnitFuncBody
			spanNode := ast.Node(decl.Body)
			if decl.Body == nil {
				// A bodyless declaration is an implementation obligation;
				// its span is the declaration span.
				kind = identity.UnitBodylessDecl
				spanNode = decl
			}
			unit, err := newSourceUnit(u, file, fset, spanNode, kind, funcDisplayName(decl), raw)
			if err != nil {
				return err
			}
			file.units = append(file.units, unit)
			if recursive && decl.Body != nil {
				if err := censusFuncLits(u, file, fset, decl.Body, funcDisplayName(decl), raw); err != nil {
					return err
				}
			}
		case *ast.GenDecl:
			if decl.Tok.String() != "var" {
				continue
			}
			for _, spec := range decl.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok || len(value.Values) == 0 {
					continue
				}
				unit, err := newSourceUnit(u, file, fset, value, identity.UnitVarInitializer, specDisplayName(value), raw)
				if err != nil {
					return err
				}
				file.units = append(file.units, unit)
				if recursive {
					if err := censusFuncLits(u, file, fset, value, specDisplayName(value), raw); err != nil {
						return err
					}
				}
			}
		}
	}
	if file.censusMode == CensusManifest {
		record, supplied := u.manifest.fileRecord(file.id)
		if !supplied {
			return &LoadError{Dir: u.request.Dir, Reason: file.id.String() +
				": provider-owned file requires a unit-manifest record and the request supplies none" +
				" (produce the toolchain audit artifact and select it on the request)"}
		}
		interiors, err := joinManifestUnits(u, file, record, raw)
		if err != nil {
			return err
		}
		file.units = append(file.units, interiors...)
	}
	sort.Slice(file.units, func(i, j int) bool {
		return file.units[i].span.Start.Offset < file.units[j].span.Start.Offset
	})
	return nil
}

// newSourceUnit builds one censused unit with its physical span and
// overlay-aware SourceSpanHash.
func newSourceUnit(u *Universe, file *LoadedFile, fset *token.FileSet, node ast.Node, kind identity.UnitKind, name string, raw []byte) (SourceUnit, error) {
	start := fset.PositionFor(node.Pos(), false)
	end := fset.PositionFor(node.End(), false)
	span := Span{
		Start: Position{Line: start.Line, Column: start.Column, Offset: start.Offset},
		End:   Position{Line: end.Line, Column: end.Column, Offset: end.Offset},
	}
	spanID, err := identity.NewSpanID(file.id, span.Start.Offset, span.End.Offset)
	if err != nil {
		return SourceUnit{}, err
	}
	unitID, err := identity.NewSourceUnitID(spanID, kind)
	if err != nil {
		return SourceUnit{}, err
	}
	if span.End.Offset > len(raw) || span.Start.Offset > span.End.Offset {
		return SourceUnit{}, &LoadError{Dir: u.request.Dir, Reason: unitID.String() + ": span exceeds source bytes"}
	}
	return SourceUnit{
		id: unitID, name: name, span: span,
		hash: sha256.Sum256(raw[span.Start.Offset:span.End.Offset]),
	}, nil
}

// censusFuncLits recursively censuses every function literal inside one
// top-level unit, in source order; the pre-scope ledger is total. Per-unit
// C-dependence is derived later from typed evidence, never inherited here.
func censusFuncLits(u *Universe, file *LoadedFile, fset *token.FileSet, root ast.Node, parentName string, raw []byte) error {
	var walkErr error
	index := 0
	ast.Inspect(root, func(n ast.Node) bool {
		if walkErr != nil {
			return false
		}
		lit, ok := n.(*ast.FuncLit)
		if !ok {
			return true
		}
		index++
		unit, err := newSourceUnit(u, file, fset, lit, identity.UnitFuncLitBody,
			fmt.Sprintf("%s$lit%d", parentName, index), raw)
		if err != nil {
			walkErr = err
			return false
		}
		file.units = append(file.units, unit)
		return true // literals nest; keep descending
	})
	return walkErr
}

func funcDisplayName(decl *ast.FuncDecl) string {
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		return "(" + recvDisplay(decl.Recv.List[0]) + ")." + decl.Name.Name
	}
	return decl.Name.Name
}

func recvDisplay(field *ast.Field) string {
	switch t := field.Type.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return "*" + ident.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return "recv"
}

func specDisplayName(spec *ast.ValueSpec) string {
	names := make([]string, 0, len(spec.Names))
	for _, name := range spec.Names {
		names = append(names, name.Name)
	}
	return strings.Join(names, ",")
}

// SelectedFileBytes reads a file's selected bytes, honoring request overlays.
// It is the single owner of selected-byte access; producers and verifiers that
// must parse exactly what resolution selected read through it, never the disk
// path directly.
func SelectedFileBytes(path string, overlay map[string][]byte) ([]byte, error) {
	return fileBytes(path, overlay)
}

// fileBytes reads a file's selected bytes, honoring overlays.
func fileBytes(path string, overlay map[string][]byte) ([]byte, error) {
	if content, overlaid := overlay[path]; overlaid {
		return content, nil
	}
	return os.ReadFile(path)
}

// parseFileBytes parses a file from its selected bytes.
func parseFileBytes(fset *token.FileSet, path string, overlay map[string][]byte) (*ast.File, error) {
	raw, err := fileBytes(path, overlay)
	if err != nil {
		return nil, err
	}
	return parser.ParseFile(fset, path, raw, parser.ParseComments|parser.SkipObjectResolution)
}

// sortLoadedFiles orders transient files deterministically.
func sortLoadedFiles(files []*LoadedFile) {
	sort.Slice(files, func(i, j int) bool { return files[i].id.Rel() < files[j].id.Rel() })
}
