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
		if err := joinCheckedDecls(u, pkg); err != nil {
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
	// C-dependence is per-unit cgo evidence: it exists only inside files that
	// import the "C" pseudo-package, and only for selector bases that
	// actually resolve to that import (a shadowing local C never classifies).
	importsC := false
	for _, imported := range syntax.Imports {
		if imported.Path != nil && imported.Path.Value == `"C"` {
			importsC = true
		}
	}
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
			unit.cDependent = importsC && referencesC(decl)
			file.units = append(file.units, unit)
			if recursive && decl.Body != nil {
				if err := censusFuncLits(u, file, fset, decl.Body, funcDisplayName(decl), importsC, unit.cDependent, raw); err != nil {
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
				unit.cDependent = importsC && referencesC(value)
				file.units = append(file.units, unit)
				if recursive {
					if err := censusFuncLits(u, file, fset, value, specDisplayName(value), importsC, unit.cDependent, raw); err != nil {
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

// joinCheckedDecls joins each checked-view declaration to its origin unit via
// //line display evidence, or records it as a typed synthetic unit. Basename
// and declaration-name matching never participate.
func joinCheckedDecls(u *Universe, pkg *LoadedPackage) error {
	if len(pkg.checkedDecls) == 0 {
		return nil
	}
	unitsByFile := map[string][]*SourceUnit{} // display path -> units
	for _, file := range pkg.files {
		for i := range file.units {
			unitsByFile[file.path] = append(unitsByFile[file.path], &file.units[i])
		}
	}
	for i := range pkg.checkedDecls {
		decl := &pkg.checkedDecls[i]
		display := u.fset.Position(decl.node.Pos()) // //line-adjusted
		physical := u.fset.PositionFor(decl.node.Pos(), false)
		endPhysical := u.fset.PositionFor(decl.node.End(), false)
		decl.span = Span{
			Start: Position{Line: physical.Line, Column: physical.Column, Offset: physical.Offset},
			End:   Position{Line: endPhysical.Line, Column: endPhysical.Column, Offset: endPhysical.Offset},
		}
		decl.name = declDisplayName(decl.node)
		origin := findOriginUnit(unitsByFile[display.Filename], display.Line)
		if origin != nil {
			decl.origin = origin.id
			pkg.mappings = append(pkg.mappings, CheckedUnitMapping{source: origin.id, checked: decl.span})
			continue
		}
		// No origin in user source: a package-synthetic checked declaration.
		// Its role derives from the declaration's own kind — never a path.
		role := SyntheticAdapter
		if generic, isGen := decl.node.(*ast.GenDecl); isGen {
			role = SyntheticData
			for _, spec := range generic.Specs {
				if _, isType := spec.(*ast.TypeSpec); isType {
					role = SyntheticTypeDecl
				}
			}
		}
		synthetic, err := NewSyntheticUnit(pkg.id, decl.name, role)
		if err != nil {
			return err
		}
		pkg.synthetics = append(pkg.synthetics, synthetic)
	}
	return nil
}

// findOriginUnit locates the unit whose physical line range contains the
// display line reported by the checked view's //line evidence.
func findOriginUnit(units []*SourceUnit, line int) *SourceUnit {
	for _, unit := range units {
		if unit.span.Start.Line <= line && line <= unit.span.End.Line {
			return unit
		}
	}
	return nil
}

// censusFuncLits recursively censuses every function literal inside one
// top-level unit, in source order; the pre-scope ledger is total.
// A literal nested within a C-dependent parent executes in that external
// context and inherits its dependence.
func censusFuncLits(u *Universe, file *LoadedFile, fset *token.FileSet, root ast.Node, parentName string, importsC, parentCDependent bool, raw []byte) error {
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
		unit.cDependent = parentCDependent || (importsC && referencesC(lit))
		file.units = append(file.units, unit)
		return true // literals nest; keep descending
	})
	return walkErr
}

// referencesC reports whether the subtree references the cgo "C"
// pseudo-package: a selector whose base identifier spells C and is not
// shadowed by any enclosing local binding (parameter, receiver, or local
// declaration) between the reference and the file scope.
func referencesC(node ast.Node) bool {
	found := false
	var walk func(n ast.Node, shadowed bool)
	walk = func(n ast.Node, shadowed bool) {
		if n == nil || found {
			return
		}
		switch n := n.(type) {
		case *ast.FuncDecl:
			walk(n.Type, shadowed)
			if n.Body != nil {
				walk(n.Body, shadowed || bindsC(n.Recv) || bindsC(n.Type.Params) || bindsC(n.Type.Results))
			}
			return
		case *ast.FuncLit:
			walk(n.Body, shadowed || bindsC(n.Type.Params) || bindsC(n.Type.Results))
			return
		case *ast.BlockStmt:
			blockShadowed := shadowed
			for _, stmt := range n.List {
				if declaresC(stmt) {
					blockShadowed = true
				}
				walk(stmt, blockShadowed)
			}
			return
		case *ast.SelectorExpr:
			if base, ok := n.X.(*ast.Ident); ok && base.Name == "C" && !shadowed {
				found = true
				return
			}
			walk(n.X, shadowed)
			return
		}
		// Generic descent preserving the current shadow state.
		ast.Inspect(n, func(child ast.Node) bool {
			if child == nil || child == n || found {
				return child == n && !found
			}
			walk(child, shadowed)
			return false
		})
	}
	walk(node, false)
	return found
}

// bindsC reports whether a field list declares an identifier C.
func bindsC(fields *ast.FieldList) bool {
	if fields == nil {
		return false
	}
	for _, field := range fields.List {
		for _, name := range field.Names {
			if name.Name == "C" {
				return true
			}
		}
	}
	return false
}

// declaresC reports whether a statement introduces a binding named C into the
// enclosing block scope.
func declaresC(stmt ast.Stmt) bool {
	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		if stmt.Tok.String() == ":=" {
			for _, lhs := range stmt.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && ident.Name == "C" {
					return true
				}
			}
		}
	case *ast.DeclStmt:
		if decl, ok := stmt.Decl.(*ast.GenDecl); ok {
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						if name.Name == "C" {
							return true
						}
					}
				case *ast.TypeSpec:
					if spec.Name.Name == "C" {
						return true
					}
				}
			}
		}
	}
	return false
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

func declDisplayName(node ast.Node) string {
	switch decl := node.(type) {
	case *ast.FuncDecl:
		return funcDisplayName(decl)
	case *ast.GenDecl:
		if len(decl.Specs) > 0 {
			switch spec := decl.Specs[0].(type) {
			case *ast.ValueSpec:
				return specDisplayName(spec)
			case *ast.TypeSpec:
				return spec.Name.Name
			}
		}
	}
	return "decl"
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
