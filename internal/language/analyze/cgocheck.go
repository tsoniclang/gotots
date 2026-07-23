package analyze

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// kindByASTName joins go/ast type names to pinned catalog kinds, independently
// of the producer's catalog-driven classifier — the same reflection-name lookup
// the source re-parse join uses, here applied to the checked cgo view.
var kindByASTName = func() map[string]catalog.Kind {
	m := map[string]catalog.Kind{}
	for _, kind := range catalog.All() {
		m[kind.Name()] = kind
	}
	return m
}()

// cgoRefKey is one cgo implementation reference, keyed identically to the
// application site<->reference join so a cgo reference is conserved on the same
// terms — never exempted.
type cgoRefKey struct {
	owner    string
	occ      string
	edge     string
	child    string
	contract uint8
	ordinal  int
	anchor   string
}

func (k cgoRefKey) String() string {
	return fmt.Sprintf("{owner=%s occ=%s edge=%s child=%s contract=%d ord=%d anchor=%s}",
		k.owner, k.occ, k.edge, k.child, k.contract, k.ordinal, k.anchor)
}

// VerifyCgoReferences independently derives the checked-view reference topology
// of every full cgo unit and exact-multiset-joins it against the producer's cgo
// references in the inventory. Cgo is NOT exempted from reference conservation:
// a source re-parse cannot reproduce the checked topology, so the independent
// derivation walks the checked counterpart AST through a reflection field walk
// (a different traversal than the catalog-driven producer), keying child and
// anchor identities from the separately certified origin map. A missing,
// duplicate, mis-parented, reordered, or edge-changed cgo reference fails here
// with exact one-sided identities.
//
// It runs on the transient universe (checked ASTs are live) after Analyze and
// before finalization severs them.
func VerifyCgoReferences(universe *source.Universe, inv *WorkspaceInventory) error {
	invByPkg := map[string]*PackageInventory{}
	for _, pkg := range inv.Packages() {
		invByPkg[pkg.ID().String()] = pkg
	}
	var problems []string
	for _, pkg := range universe.Packages() {
		if pkg.Disposition() != source.DispositionOrdinarySource {
			continue
		}
		counterparts := pkg.CgoCounterparts()
		if len(counterparts) == 0 {
			continue
		}
		pkgInv := invByPkg[pkg.ID().String()]
		if pkgInv == nil {
			continue
		}
		// Cgo units: source units whose file is a cgo original.
		cgoUnit := map[identity.SourceUnitID]bool{}
		for _, file := range pkg.Files() {
			if !file.CgoOriginal() {
				continue
			}
			for _, unit := range file.Units() {
				cgoUnit[unit.ID()] = true
			}
		}
		// Full cgo units own a body region (RootUnit set); only those are walked
		// by the producer, so only those are independently derived here — a
		// non-full cgo unit contributes no body region and no reference.
		hasBodyRegion := map[identity.SourceUnitID]bool{}
		for _, region := range pkgInv.Files() {
			if root := region.RootUnit(); !root.IsZero() {
				hasBodyRegion[root] = true
			}
		}
		// Producer cgo references: those whose parent owner is a cgo unit body.
		producer := map[cgoRefKey]int{}
		for _, ref := range pkgInv.References() {
			owner := ref.Parent()
			if owner.IsFileDeclaration() || owner.IsPackageInitialization() {
				continue
			}
			if !cgoUnit[owner.Unit().Source()] {
				continue
			}
			producer[cgoRefKey{
				owner: owner.String(), occ: ref.ParentOccurrence().String(), edge: ref.Edge().String(),
				child: ref.Child().String(), contract: uint8(ref.Contract()),
				ordinal: ref.Ordinal(), anchor: ref.Anchor().String(),
			}]++
		}
		// Independent derivation over every full cgo unit's checked counterpart.
		derived := map[cgoRefKey]int{}
		for _, file := range pkg.Files() {
			if !file.CgoOriginal() {
				continue
			}
			for _, unit := range file.Units() {
				if !hasBodyRegion[unit.ID()] {
					continue // non-full cgo unit: no body region, no references
				}
				node, _, ok := pkg.CgoCounterpartNode(unit.ID())
				if !ok {
					continue
				}
				d := &cgoRefDeriver{
					fset: universe.Fset(), file: unit.ID().Span().File(),
					boundaries: counterparts, owner: "unit:" + unit.ID().String(),
					ordinals: map[string]int{}, refs: derived,
				}
				d.walk(node, nil, "")
				if d.err != nil {
					problems = append(problems, d.err.Error())
				}
			}
		}
		for key, n := range derived {
			if producer[key] != n {
				problems = append(problems, fmt.Sprintf("%s: independently derived cgo reference %s x%d, inventory has %d", pkg.ID(), key, n, producer[key]))
			}
		}
		for key, n := range producer {
			if derived[key] != n {
				problems = append(problems, fmt.Sprintf("%s: inventory cgo reference %s x%d, independent derivation has %d", pkg.ID(), key, n, derived[key]))
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return &ConsumeError{Reason: "cgo reference conservation failed; " + fmt.Sprint(problems)}
	}
	return nil
}

// cgoRefDeriver independently walks one checked cgo counterpart, emitting a
// reference at each nested-unit boundary (a node in the certified origin map).
type cgoRefDeriver struct {
	fset       *token.FileSet
	file       identity.FileID // original cgo file identity (offsets are checked coordinates)
	boundaries map[ast.Node]identity.SourceUnitID
	owner      string
	ordinals   map[string]int
	refs       map[cgoRefKey]int
	err        error
}

// walk descends node; parent/field name the edge from its direct parent. At a
// boundary (a checked node the origin map maps to an original unit) it records a
// reference — with original child/anchor identities and checked-coordinate
// parent occurrence — and does not descend.
func (d *cgoRefDeriver) walk(node ast.Node, parent ast.Node, field string) {
	if d.err != nil || node == nil {
		return
	}
	if parent != nil {
		if child, ok := d.boundaries[node]; ok {
			contract, err := ContractForKind(child.Kind())
			if err != nil {
				d.err = err
				return
			}
			d.refs[cgoRefKey{
				owner:    d.owner,
				occ:      d.parentOcc(parent),
				edge:     astTypeName(parent) + "." + field,
				child:    child.String(),
				contract: uint8(contract),
				ordinal:  d.ordinals[d.owner],
				anchor:   child.Span().String(),
			}]++
			d.ordinals[d.owner]++
			return
		}
	}
	d.descend(node)
}

// parentOcc is the occurrence identity of the direct parent node: original file
// identity with checked-coordinate offsets and the catalog kind, matching the
// producer's cgo occurrence identity exactly.
func (d *cgoRefDeriver) parentOcc(parent ast.Node) string {
	kind, known := kindByASTName[astTypeName(parent)]
	if !known {
		return ""
	}
	start := d.fset.PositionFor(parent.Pos(), false).Offset
	end := d.fset.PositionFor(parent.End(), false).Offset
	span, err := identity.NewSpanID(d.file, start, end)
	if err != nil {
		return ""
	}
	occ, err := identity.NewOccurrenceID(span, uint16(kind))
	if err != nil {
		return ""
	}
	return occ.String()
}

// descend visits each child slot via reflection, naming its field for the edge —
// a different traversal than the producer's catalog-driven descent.
func (d *cgoRefDeriver) descend(node ast.Node) {
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
		d.descendValue(elem.Field(i), node, typ.Field(i).Name)
	}
}

func (d *cgoRefDeriver) descendValue(fv reflect.Value, parent ast.Node, field string) {
	switch fv.Kind() {
	case reflect.Interface, reflect.Pointer:
		if fv.IsNil() {
			return
		}
		if child, ok := fv.Interface().(ast.Node); ok {
			d.walk(child, parent, field)
		}
	case reflect.Slice:
		for j := 0; j < fv.Len(); j++ {
			d.descendValue(fv.Index(j), parent, field)
		}
	}
}

// astTypeName is the go/ast type name of a node (its catalog-spelled kind name).
func astTypeName(node ast.Node) string {
	name := reflect.TypeOf(node).String()
	return name[len("*ast."):]
}
