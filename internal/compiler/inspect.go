// Package compiler sequences the compilation phases. It holds no semantic
// cases: it orders the phase owners, runs each stage's blocking independent
// verifier before any downstream stage consumes the artifact, and surfaces
// typed results. cmd/gotots is its only caller; there is exactly one
// compilation route — inspection uses the same request and loader as
// generation will.
package compiler

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
	"github.com/tsoniclang/gotots/internal/stagecheck"
)

// Inspection is the verified result of one inspect run: the finalized source
// universe (identity, provenance, acquisition, versions, evidence-depth
// partition) and the construct inventory of the full-semantic scope.
type Inspection struct {
	workspace *source.Workspace
	selection *scope.Selection
	inventory *analyze.WorkspaceInventory
}

// Workspace is the finalized source universe.
func (i *Inspection) Workspace() *source.Workspace { return i.workspace }

// Selection is the scope phase's immutable evidence-depth selection.
func (i *Inspection) Selection() *scope.Selection { return i.selection }

// Inventory is the verified construct inventory.
func (i *Inspection) Inventory() *analyze.WorkspaceInventory { return i.inventory }

// InspectConstructs resolves a compilation request into a verified
// whole-workspace construct inventory:
//
//	request -> ResolveContract -> [decode audit/manifest artifact]
//	        -> LoadUniverse(policy, manifest) -> scope.Select(contract)
//	        -> Finalize -> [verify universe] -> [verify unit census]
//	        -> [verify audit context] -> inventory (full-semantic scope)
//	        -> [verify inventory] -> report
//
// The contract resolves before any policy-dependent census or acquisition
// decision. A failed stage verifier blocks every downstream stage; there is
// no partial or unverified artifact.
func InspectConstructs(req source.Request) (*Inspection, error) {
	contract, err := scope.ResolveContract(req.ProviderContract, req.ProviderContractDigest, req.ProviderContractArtifact)
	if err != nil {
		return nil, err
	}
	policy, err := contract.AcquisitionPolicy()
	if err != nil {
		return nil, err
	}
	var artifact *analyze.AuditArtifact
	manifest := source.UnitManifest{}
	if req.AuditArtifact != "" {
		artifact, err = analyze.DecodeAuditArtifactBound(req.AuditArtifact, req.AuditArtifactDigest)
		if err != nil {
			return nil, err
		}
		manifest, err = manifestFromArtifact(artifact)
		if err != nil {
			return nil, err
		}
	}
	universe, err := source.LoadUniverse(req, policy, manifest)
	if err != nil {
		return nil, err
	}
	selection, err := scope.Select(universe, contract)
	if err != nil {
		return nil, err
	}
	// The analyze traversal runs on the transient checker graph, after scope
	// selection and before source finalization; source finalization then
	// consumes only the traversal's opaque retention projection.
	inventory, projection, err := analyze.Analyze(universe, selection.Depths(), selection.ImplicitDepths())
	if err != nil {
		return nil, err
	}
	ws, err := source.Finalize(universe, selection.Depths(), selection.ImplicitDepths(), projection)
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifySourceUniverse(ws, req); err != nil {
		return nil, err
	}
	if err := stagecheck.VerifyUnitCensus(ws, req, contract, policy); err != nil {
		return nil, err
	}
	if artifact != nil {
		if err := analyze.VerifyAuditContext(ws, auditMeta(req, contract), artifact); err != nil {
			return nil, err
		}
	}
	if err := stagecheck.VerifyInventory(req, ws, inventory); err != nil {
		return nil, err
	}
	return &Inspection{workspace: ws, selection: selection, inventory: inventory}, nil
}

// manifestFromArtifact converts a decoded audit artifact's per-file records
// into the loader's unit-manifest input.
func manifestFromArtifact(artifact *analyze.AuditArtifact) (source.UnitManifest, error) {
	files := make(map[string]source.ManifestFileRecord, len(artifact.Files))
	for _, file := range artifact.Files {
		record := source.ManifestFileRecord{Digest: file.Digest}
		for _, unit := range file.Units {
			record.Units = append(record.Units, source.ManifestUnitRecord{
				Unit: unit.Unit, Kind: unit.Kind, Start: unit.Start, End: unit.End,
				Name: unit.Name, Hash: unit.Hash, CDependent: unit.CDependent,
			})
		}
		files[file.File] = record
	}
	return source.NewUnitManifest(files)
}

// AuditCatalog resolves the request under the manifest-producing acquisition
// policy (every file censused recursively) and produces the versioned
// catalog-audit and unit-manifest artifact over the non-full closure. This is
// the toolchain-contract gate run — the producer of the artifact ordinary
// compilation consumes.
func AuditCatalog(req source.Request) (*analyze.AuditArtifact, error) {
	contract, err := scope.ResolveContract(req.ProviderContract, req.ProviderContractDigest, req.ProviderContractArtifact)
	if err != nil {
		return nil, err
	}
	auditPolicy, err := contract.AuditAcquisitionPolicy()
	if err != nil {
		return nil, err
	}
	ordinaryPolicy, err := contract.AcquisitionPolicy()
	if err != nil {
		return nil, err
	}
	universe, err := source.LoadUniverse(req, auditPolicy, source.UnitManifest{})
	if err != nil {
		return nil, err
	}
	selection, err := scope.Select(universe, contract)
	if err != nil {
		return nil, err
	}
	// The audit is a streaming gate run, not an application build: it does not
	// retain the region model. The retention projection is the selection's own
	// full-unit membership (the seal), so finalization stays a lifecycle seal.
	projection, err := selectionProjection(selection)
	if err != nil {
		return nil, err
	}
	ws, err := source.Finalize(universe, selection.Depths(), selection.ImplicitDepths(), projection)
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifySourceUniverse(ws, req); err != nil {
		return nil, err
	}
	if err := stagecheck.VerifyUnitCensus(ws, req, contract, auditPolicy); err != nil {
		return nil, err
	}
	// The audit is the provider-graph producer: extract each file's
	// definition/reference topology from the transient universe and embed it,
	// so ordinary compilation exact-joins the certified graph.
	graph, err := analyze.ExtractProviderGraph(universe)
	if err != nil {
		return nil, err
	}
	artifact, err := analyze.AuditCatalog(ws, auditMeta(req, contract), req.Overlay, ordinaryPolicy, graph)
	if err != nil {
		return nil, err
	}
	// Certify the embedded provider graph independently before the artifact's
	// digest is trusted: re-derive every audited file's site set from source and
	// exact-join it against the embedded references.
	if err := stagecheck.VerifyProviderGraph(ws, artifact.Files, req.Overlay); err != nil {
		return nil, err
	}
	return artifact, nil
}

// selectionProjection builds the retention projection directly from the scope
// selection's full-unit membership, for gate runs that do not build the region
// model.
func selectionProjection(selection *scope.Selection) (source.RetentionProjection, error) {
	var fullUnits []identity.SourceUnitID
	for _, unit := range selection.Units() {
		if unit.Depth == source.DepthFullSemantic {
			fullUnits = append(fullUnits, unit.Unit)
		}
	}
	var fullImplicit []identity.ImplicitUnitID
	for _, unit := range selection.ImplicitUnits() {
		if unit.Depth == source.DepthFullSemantic {
			fullImplicit = append(fullImplicit, unit.Unit)
		}
	}
	return source.NewRetentionProjection(fullUnits, fullImplicit)
}

// auditMeta binds an audit artifact to the request's production context.
func auditMeta(req source.Request, contract scope.ProviderContract) analyze.AuditMeta {
	return analyze.AuditMeta{
		ContractID:          contract.ID(),
		ContractFingerprint: contract.Fingerprint(),
		BuildFlags:          strings.Join(req.BuildFlags, " "),
		OverlayDigest:       overlayDigest(req.Overlay),
	}
}

// overlayDigest canonically digests the request overlays.
func overlayDigest(overlay map[string][]byte) string {
	if len(overlay) == 0 {
		return ""
	}
	paths := make([]string, 0, len(overlay))
	for path := range overlay {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, path := range paths {
		fmt.Fprintf(h, "%s=%x|", path, sha256.Sum256(overlay[path]))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// VerifyAuditArtifact is the verification of a stored artifact against one
// inspection's universe and request context: internal invariants, production
// context, and exact membership with per-file digest and unit joins.
func VerifyAuditArtifact(inspection *Inspection, req source.Request, path string) error {
	contract, err := scope.ResolveContract(req.ProviderContract, req.ProviderContractDigest, req.ProviderContractArtifact)
	if err != nil {
		return err
	}
	ordinary, err := contract.AcquisitionPolicy()
	if err != nil {
		return err
	}
	return analyze.VerifyAuditArtifact(inspection.workspace, auditMeta(req, contract), path, ordinary)
}

// AuditVerify is the gate coverage check: it resolves the request's universe
// afresh under the manifest-producing (recursive) policy — so every provider
// interior is independently derived — and exact-joins the stored artifact
// bidirectionally, including per-file unit manifests. A manifest that omits,
// fabricates, or mutates one unit fails here even when correctly sealed.
func AuditVerify(req source.Request, path string) error {
	contract, err := scope.ResolveContract(req.ProviderContract, req.ProviderContractDigest, req.ProviderContractArtifact)
	if err != nil {
		return err
	}
	auditPolicy, err := contract.AuditAcquisitionPolicy()
	if err != nil {
		return err
	}
	ordinaryPolicy, err := contract.AcquisitionPolicy()
	if err != nil {
		return err
	}
	universe, err := source.LoadUniverse(req, auditPolicy, source.UnitManifest{})
	if err != nil {
		return err
	}
	selection, err := scope.Select(universe, contract)
	if err != nil {
		return err
	}
	projection, err := selectionProjection(selection)
	if err != nil {
		return err
	}
	ws, err := source.Finalize(universe, selection.Depths(), selection.ImplicitDepths(), projection)
	if err != nil {
		return err
	}
	return analyze.VerifyAuditArtifact(ws, auditMeta(req, contract), path, ordinaryPolicy)
}
