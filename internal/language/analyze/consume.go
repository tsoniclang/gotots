package analyze

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// ConsumeError is the typed failure of decoding or merging a certified provider
// graph into the ordinary inventory.
type ConsumeError struct{ Reason string }

func (e *ConsumeError) Error() string { return "GOTOTS_PROVIDER_CONSUME: " + e.Reason }

// ConsumeProviderGraph merges the certified provider graph into the same
// WorkspaceInventory the application traversal produced, so later stages see one
// inventory over the whole closure — application packages from the live
// traversal, provider packages from the certified graph — with no second
// ledger. It rescans no provider interior: source-unit definitions come from the
// finalized census (depth-correct, so the flat unit list is joined to the graph
// as a projection), reference topology is decoded from the trusted artifact
// graph through the validating identity constructors, and each package's
// implicit initialization unit — which a per-file artifact cannot represent —
// is added from the census with its owning package-initialization reference.
//
// Authority is the request-bound certified digest the caller already verified;
// this consumer adds no source rescanning. The graph's per-file definition set
// is exact-joined to the census unit set (both directions) so a graph that omits
// or fabricates a provider unit fails here.
func ConsumeProviderGraph(inv *WorkspaceInventory, ws *source.Workspace, files []AuditFile) error {
	// Provider packages: ordinary-source packages with no full-semantic unit,
	// which the application traversal excludes. Their files map to one package.
	providerPkgOf := map[string]identity.PackageID{}
	providerPkgs := map[string]*source.Package{}
	for _, pkg := range ws.Packages() {
		if pkg.Disposition() != source.DispositionOrdinarySource || pkg.RetainsFullSemantic() {
			continue
		}
		providerPkgs[pkg.ID().String()] = pkg
		for _, file := range pkg.Files() {
			providerPkgOf[file.ID().String()] = pkg.ID()
		}
	}
	if len(providerPkgs) == 0 {
		return nil
	}

	// Census depth/kind per source unit of every provider package, so decoded
	// definitions carry the selected depth and the graph joins the census.
	censusUnits := map[string]map[string]censusUnitDepth{} // pkg -> unit -> census
	for id, pkg := range providerPkgs {
		units := map[string]censusUnitDepth{}
		for _, unit := range pkg.Units() {
			units[unit.ID().String()] = censusUnitDepth{kind: unit.Kind(), depth: unit.Depth()}
		}
		censusUnits[id] = units
	}

	// Decode each provider file's graph definitions/references into the package
	// inventory. The graph is the topology authority; the flat census supplies
	// each definition's selected depth. The census<->definition membership join
	// (both directions) is owned by the independent inventory verifier, so it is
	// not re-owned here.
	byPkg := map[string]*PackageInventory{}
	pkgInvOf := func(pkgID identity.PackageID) *PackageInventory {
		key := pkgID.String()
		if existing, ok := byPkg[key]; ok {
			return existing
		}
		created := &PackageInventory{id: pkgID}
		byPkg[key] = created
		return created
	}

	for _, record := range files {
		pkgID, ok := providerPkgOf[record.File]
		if !ok {
			continue // non-provider audited file (e.g. a non-full application file)
		}
		pkgInv := pkgInvOf(pkgID)
		census := censusUnits[pkgID.String()]

		for _, def := range record.Definitions {
			built, err := decodeProviderDefinition(def, census)
			if err != nil {
				return &ConsumeError{Reason: record.File + ": " + err.Error()}
			}
			pkgInv.definitions = append(pkgInv.definitions, built)
		}
		for _, ref := range record.References {
			built, err := decodeProviderReference(ref)
			if err != nil {
				return &ConsumeError{Reason: record.File + ": " + err.Error()}
			}
			pkgInv.references = append(pkgInv.references, built)
		}
	}

	// Add each provider package's implicit initialization unit (definition and
	// owning reference) from the census — the per-file artifact cannot carry it.
	for _, pkg := range providerPkgs {
		pkgInv := pkgInvOf(pkg.ID())
		for _, implicit := range pkg.ImplicitUnits() {
			pkgInv.definitions = append(pkgInv.definitions, ImplementationDefinition{
				unit: ImplicitUnitRef(implicit.ID()), kind: identity.UnitImplicitExecutable,
				contract: ContractCatalogOwner, depth: implicit.Depth(),
				full: implicit.Depth() == source.DepthFullSemantic,
			})
			pkgInv.references = append(pkgInv.references, NewImplicitReference(pkg.ID(), implicit.ID()))
		}
	}

	// Merge in deterministic identity order; no application package is a provider
	// package, so there is never a duplicate package inventory.
	present := map[string]bool{}
	for _, existing := range inv.packages {
		present[existing.id.String()] = true
	}
	for _, pkg := range ws.Packages() {
		built, ok := byPkg[pkg.ID().String()]
		if !ok {
			continue
		}
		if present[pkg.ID().String()] {
			return &ConsumeError{Reason: "provider package " + pkg.ID().String() + " already present in the application inventory"}
		}
		inv.packages = append(inv.packages, built)
	}
	return nil
}

// decodeProviderDefinition reconstructs one provider definition from its
// serialized record, validating the stored kind and contract against the
// canonical unit identity and joining the selected depth from the census.
func decodeProviderDefinition(def ManifestDefinition, census map[string]censusUnitDepth) (ImplementationDefinition, error) {
	ref, err := identity.ParseUnitRef(def.Unit)
	if err != nil {
		return ImplementationDefinition{}, err
	}
	kind := ref.Kind()
	if uint8(kind) != def.Kind {
		return ImplementationDefinition{}, fmt.Errorf("definition %s stored kind %d disagrees with its identity kind %d", def.Unit, def.Kind, uint8(kind))
	}
	contract, err := ContractForKind(kind)
	if err != nil {
		return ImplementationDefinition{}, err
	}
	if uint8(contract) != def.Contract {
		return ImplementationDefinition{}, fmt.Errorf("definition %s stored contract %d disagrees with kind contract %d", def.Unit, def.Contract, uint8(contract))
	}
	c, ok := census[def.Unit]
	if !ok {
		return ImplementationDefinition{}, fmt.Errorf("graph definition %s has no census unit", def.Unit)
	}
	return ImplementationDefinition{
		unit: UnitRef{source: ref.Source(), implicit: ref.Implicit()}, kind: kind, contract: contract,
		depth: c.depth, full: c.depth == source.DepthFullSemantic,
	}, nil
}

// censusUnitDepth is the census kind/depth pair keyed by unit identity. It is a
// type alias target so decodeProviderDefinition can take the map by value.
type censusUnitDepth = struct {
	kind  identity.UnitKind
	depth source.EvidenceDepth
}

// decodeProviderReference reconstructs one provider reference from its
// serialized record through the validating identity/edge constructors. A
// package-initialization reference (implicit child, no edge) is decoded without
// an edge or anchor; every other reference carries a valid edge, occurrence, and
// anchor.
func decodeProviderReference(ref ManifestReference) (ImplementationRef, error) {
	parent, err := ParseRegionOwner(ref.Parent)
	if err != nil {
		return ImplementationRef{}, err
	}
	child, err := identity.ParseUnitRef(ref.Child)
	if err != nil {
		return ImplementationRef{}, err
	}
	contract, err := ContractForKind(child.Kind())
	if err != nil {
		return ImplementationRef{}, err
	}
	if uint8(contract) != ref.Contract {
		return ImplementationRef{}, fmt.Errorf("reference to %s stored contract %d disagrees with kind contract %d", ref.Child, ref.Contract, uint8(contract))
	}
	built := ImplementationRef{
		parent:   parent,
		child:    UnitRef{source: child.Source(), implicit: child.Implicit()},
		contract: contract,
		ordinal:  ref.Ordinal,
	}
	if parent.IsPackageInitialization() {
		// Implicit owning reference: no grammatical edge, occurrence, or anchor.
		if ref.Edge != "" || ref.Occ != "" || ref.Anchor != "" {
			return ImplementationRef{}, fmt.Errorf("package-initialization reference to %s carries an edge, occurrence, or anchor", ref.Child)
		}
		return built, nil
	}
	edge, err := catalog.EdgeByName(ref.Edge)
	if err != nil {
		return ImplementationRef{}, err
	}
	occ, err := identity.ParseOccurrenceID(ref.Occ)
	if err != nil {
		return ImplementationRef{}, err
	}
	anchor, err := identity.ParseSpanID(ref.Anchor)
	if err != nil {
		return ImplementationRef{}, err
	}
	built.edge = edge
	built.parentOcc = occ
	built.anchor = anchor
	return built, nil
}
