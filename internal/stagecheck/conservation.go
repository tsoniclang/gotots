package stagecheck

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// kindByName joins go/ast type names (the edge string's parent component) to
// pinned catalog kinds, independently of the analyze producer's classifier.
var kindByName = func() map[string]catalog.Kind {
	m := map[string]catalog.Kind{}
	for _, kind := range catalog.All() {
		m[kind.Name()] = kind
	}
	return m
}()

// siteKey is one implementation site, keyed by parent owner, grammatical edge,
// child unit, source ordinal, and anchor — the exact reference identity. It is
// derived independently from re-parsed source and exact-multiset-joined against
// the inventory's references.
type siteKey struct {
	owner   string // "decl:<fileID>" or "unit:<unitID>"
	occ     string // parent occurrence identity
	edge    string // "<ParentKind>.<Field>"
	child   string // child unit identity
	ordinal int
	anchor  string // child root span identity
}

func (s siteKey) String() string {
	return fmt.Sprintf("{owner=%s occ=%s edge=%s child=%s ord=%d anchor=%s}", s.owner, s.occ, s.edge, s.child, s.ordinal, s.anchor)
}

// verifyReferenceConservation independently extracts every application
// implementation site from re-parsed selected source and exact-multiset-joins
// the site set against the inventory's references. A missing, duplicate,
// mis-parented, reordered, or edge-changed reference fails with exact one-sided
// identities. The extraction here is a different walk than the analyze
// producer's — a reflection field walk keyed by span identity.
func verifyReferenceConservation(pkg *source.Package, refs []analyze.ImplementationRef, overlay map[string][]byte) []string {
	var problems []string
	// A reference is recorded only where the enclosing region is built: the
	// file declaration region (always, for an application file) or a full unit's
	// body region. A site inside a non-full parent is covered by the non-full
	// unit's zero-body contract, not by a reference.
	fullOwner := map[string]bool{}
	for _, file := range pkg.Files() {
		for _, unit := range file.Units() {
			if unit.Depth() == source.DepthFullSemantic {
				fullOwner["unit:"+unit.ID().String()] = true
			}
		}
	}
	builtRegion := func(owner string) bool {
		return len(owner) > 5 && owner[:5] == "decl:" || fullOwner[owner]
	}
	// Cgo references come from the checked view and cannot be reproduced by a
	// source re-parse; they are certified through the source origin cross-check,
	// so they are excluded from this source-derived join (both sides).
	cgoFile := map[string]bool{}
	for _, file := range pkg.Files() {
		if file.CgoOriginal() {
			cgoFile[file.ID().String()] = true
		}
	}
	isCgoRef := func(ref analyze.ImplementationRef) bool {
		child := ref.Child().Source()
		return !child.IsZero() && cgoFile[child.Span().File().String()]
	}
	// Inventory references, as a multiset of site keys.
	inventory := map[siteKey]int{}
	for _, ref := range refs {
		if isCgoRef(ref) {
			continue
		}
		inventory[siteKey{
			owner: ref.Parent().String(), occ: ref.ParentOccurrence().String(), edge: ref.Edge().String(),
			child: ref.Child().String(), ordinal: ref.Ordinal(), anchor: ref.Anchor().String(),
		}]++
	}
	// Independently derived sites whose enclosing region is built.
	derived := map[siteKey]int{}
	for _, file := range pkg.Files() {
		sites, err := deriveSites(file, overlay)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		for _, s := range sites {
			if builtRegion(s.owner) {
				derived[s]++
			}
		}
	}
	for key, n := range derived {
		if inventory[key] != n {
			problems = append(problems, fmt.Sprintf("site %s: independently derived %d, inventory has %d", key, n, inventory[key]))
		}
	}
	for key, n := range inventory {
		if derived[key] != n {
			problems = append(problems, fmt.Sprintf("reference %s: inventory has %d, independent derivation has %d", key, n, derived[key]))
		}
	}
	sort.Strings(problems)
	return problems
}

// deriveSites re-parses one file's selected bytes and independently derives its
// implementation sites: the enclosing edge of every function body, function
// literal, and initialized value spec. A cgo original (no re-derivable
// checked-view topology from source alone) contributes no sites here; its
// references are certified through the source origin cross-check.
func deriveSites(file *source.File, overlay map[string][]byte) ([]siteKey, error) {
	raw, overlaid := overlay[file.Path()]
	if !overlaid {
		var err error
		raw, err = os.ReadFile(file.Path())
		if err != nil {
			return nil, fmt.Errorf("%s: unreadable: %v", file.ID(), err)
		}
	}
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, file.Path(), raw, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("%s: unparsable: %v", file.ID(), err)
	}
	if file.CgoOriginal() {
		return nil, nil
	}
	d := &siteDeriver{fset: fset, fileID: file.ID(), ordinals: map[string]int{}}
	d.walk(syntax, nil, "", "decl:"+file.ID().String())
	return d.sites, d.err
}

// siteDeriver walks one file tracking the enclosing region owner and emits a
// site at each unit boundary.
type siteDeriver struct {
	fset     *token.FileSet
	fileID   identity.FileID
	sites    []siteKey
	ordinals map[string]int // owner -> next ordinal
	err      error
}

// walk descends node; parent/field name the edge from its direct parent; owner
// is the nearest enclosing unit region (or the file declaration region). A unit
// boundary is detected at the exact (parent, field, node) triple analyze uses.
func (d *siteDeriver) walk(node ast.Node, parent ast.Node, field, owner string) {
	if d.err != nil || node == nil {
		return
	}
	childOwner := owner
	if unitID, ok := d.unitBoundary(parent, field, node, owner); ok {
		d.sites = append(d.sites, siteKey{
			owner:   owner,
			occ:     d.parentOcc(parent),
			edge:    astKind(parent) + "." + field,
			child:   unitID.String(),
			ordinal: d.ordinals[owner],
			anchor:  unitID.Span().String(),
		})
		d.ordinals[owner]++
		childOwner = "unit:" + unitID.String() // this unit's own body region
	}
	d.descend(node, childOwner)
}

// unitBoundary reports whether the child node at (parent, field) is an
// implementation-unit root, returning its identity. Detection matches analyze's
// unit-root recording exactly: a function declaration's body block, a bodyless
// function declaration, any function literal, and an initialized package value
// spec.
func (d *siteDeriver) unitBoundary(parent ast.Node, field string, node ast.Node, owner string) (identity.SourceUnitID, bool) {
	switch n := node.(type) {
	case *ast.BlockStmt:
		if fd, ok := parent.(*ast.FuncDecl); ok && field == "Body" && fd.Body == n {
			return d.unitID(n, identity.UnitFuncBody)
		}
	case *ast.FuncDecl:
		if n.Body == nil {
			return d.unitID(n, identity.UnitBodylessDecl)
		}
	case *ast.FuncLit:
		return d.unitID(n, identity.UnitFuncLitBody)
	case *ast.ValueSpec:
		// Only PACKAGE-LEVEL value specs are initializer units — a var spec
		// whose enclosing region is the file declaration region. A local var
		// inside a function body (owner is a unit region) is not a unit.
		if gd, ok := parent.(*ast.GenDecl); ok && gd.Tok == token.VAR && len(n.Values) > 0 &&
			len(owner) >= 5 && owner[:5] == "decl:" {
			return d.unitID(n, identity.UnitVarInitializer)
		}
	}
	return identity.SourceUnitID{}, false
}

// unitID builds one unit identity from a node's physical span.
func (d *siteDeriver) unitID(n ast.Node, kind identity.UnitKind) (identity.SourceUnitID, bool) {
	start := d.fset.PositionFor(n.Pos(), false).Offset
	end := d.fset.PositionFor(n.End(), false).Offset
	spanID, err := identity.NewSpanID(d.fileID, start, end)
	if err != nil {
		d.err = err
		return identity.SourceUnitID{}, false
	}
	id, err := identity.NewSourceUnitID(spanID, kind)
	if err != nil {
		d.err = err
		return identity.SourceUnitID{}, false
	}
	return id, true
}

// parentOcc is the occurrence identity of the direct parent node (span plus
// pinned catalog kind), matching the reference's parent occurrence; empty when
// the parent is the file root.
func (d *siteDeriver) parentOcc(parent ast.Node) string {
	if parent == nil {
		return ""
	}
	kind, known := kindByName[astKind(parent)]
	if !known {
		return ""
	}
	start := d.fset.PositionFor(parent.Pos(), false).Offset
	end := d.fset.PositionFor(parent.End(), false).Offset
	spanID, err := identity.NewSpanID(d.fileID, start, end)
	if err != nil {
		return ""
	}
	occ, err := identity.NewOccurrenceID(spanID, uint16(kind))
	if err != nil {
		return ""
	}
	return occ.String()
}

// descend visits each child slot, naming its field for the edge.
func (d *siteDeriver) descend(node ast.Node, owner string) {
	v := reflect.ValueOf(node)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return
	}
	typ := elem.Type()
	for i := 0; i < elem.NumField(); i++ {
		fieldName := typ.Field(i).Name
		fv := elem.Field(i)
		d.descendValue(fv, node, fieldName, owner)
	}
}

func (d *siteDeriver) descendValue(fv reflect.Value, parent ast.Node, field, owner string) {
	switch fv.Kind() {
	case reflect.Interface, reflect.Pointer:
		if fv.IsNil() {
			return
		}
		if child, ok := fv.Interface().(ast.Node); ok {
			d.walk(child, parent, field, owner)
		}
	case reflect.Slice:
		for j := 0; j < fv.Len(); j++ {
			d.descendValue(fv.Index(j), parent, field, owner)
		}
	}
}

// astKind is the catalog-spelled kind name of a node (its go/ast type name),
// matching the edge string's parent component. The empty node (a unit rooted at
// the file) yields "File".
func astKind(node ast.Node) string {
	if node == nil {
		return "File"
	}
	name := reflect.TypeOf(node).String()
	return name[len("*ast."):]
}
