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
//	request -> LoadUniverse -> scope.Select(contract) -> Finalize
//	        -> [verify universe] -> inventory (full-semantic scope)
//	        -> [verify inventory] -> report
//
// A failed stage verifier blocks every downstream stage; there is no partial
// or unverified artifact.
func InspectConstructs(req source.Request) (*Inspection, error) {
	universe, err := source.LoadUniverse(req)
	if err != nil {
		return nil, err
	}
	contract, err := scope.ResolveContract(req.ProviderContract, req.ProviderContractDigest)
	if err != nil {
		return nil, err
	}
	selection, err := scope.Select(universe, contract)
	if err != nil {
		return nil, err
	}
	ws, err := source.Finalize(universe, selection.Depths())
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifySourceUniverse(ws, req); err != nil {
		return nil, err
	}
	if err := stagecheck.VerifyUnitCensus(ws, req, contract); err != nil {
		return nil, err
	}
	inventory, err := analyze.BuildWorkspaceInventory(ws)
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifySyntaxInventory(ws, inventory); err != nil {
		return nil, err
	}
	return &Inspection{workspace: ws, selection: selection, inventory: inventory}, nil
}

// AuditCatalog resolves the request and produces the versioned catalog-audit
// artifact over the non-full closure (a toolchain-contract gate run, not part
// of ordinary compilation).
func AuditCatalog(req source.Request) (*analyze.AuditArtifact, error) {
	universe, err := source.LoadUniverse(req)
	if err != nil {
		return nil, err
	}
	contract, err := scope.ResolveContract(req.ProviderContract, req.ProviderContractDigest)
	if err != nil {
		return nil, err
	}
	selection, err := scope.Select(universe, contract)
	if err != nil {
		return nil, err
	}
	ws, err := source.Finalize(universe, selection.Depths())
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifySourceUniverse(ws, req); err != nil {
		return nil, err
	}
	return analyze.AuditCatalog(ws, auditMeta(req, contract))
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

// VerifyAuditArtifact exact-joins a stored audit artifact against one
// inspection's universe and request context.
func VerifyAuditArtifact(inspection *Inspection, req source.Request, path string) error {
	contract, err := scope.ResolveContract(req.ProviderContract, req.ProviderContractDigest)
	if err != nil {
		return err
	}
	return analyze.VerifyAuditArtifact(inspection.workspace, auditMeta(req, contract), path)
}
