// Package compiler owns pipeline sequencing only. Semantic and structural
// decisions remain in their phase owners, and every produced artifact passes
// its blocking independent verifier before downstream consumption.
package compiler

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
	"github.com/tsoniclang/gotots/internal/stagecheck"
)

// Inspection is the immutable verified Stage-1 result.
type Inspection struct {
	workspace  *source.Workspace
	plan       *sourceplan.Plan
	graph      *structure.Graph
	facts      *selectionfacts.Artifact
	selections *scope.DefinitionSelections
	executable *executable.Inventory
	hydration  source.HydrationStats
}

func (i *Inspection) Workspace() *source.Workspace             { return i.workspace }
func (i *Inspection) SourcePlan() *sourceplan.Plan             { return i.plan }
func (i *Inspection) Structure() *structure.Graph              { return i.graph }
func (i *Inspection) SelectionFacts() *selectionfacts.Artifact { return i.facts }
func (i *Inspection) Selections() *scope.DefinitionSelections  { return i.selections }
func (i *Inspection) Executable() *executable.Inventory        { return i.executable }
func (i *Inspection) Hydration() source.HydrationStats         { return i.hydration }

// InspectConstructs executes the sole ordinary Stage-1 route:
//
//	resolve contract -> resolve metadata closure
//	-> plan local/certified structural sources
//	-> hydrate one selective transient checker graph -> structural graph
//	-> closed selection facts -> definition selections
//	-> full-only executable regions -> independent transient verification
//	-> source finalization -> independent source-universe verification.
func InspectConstructs(req source.Request) (*Inspection, error) {
	selected, err := contract.Resolve(
		req.ProviderContract,
		req.ProviderContractDigest,
		req.ProviderContractArtifact,
	)
	if err != nil {
		return nil, err
	}
	certified, err := decodeSelectedArtifact(req)
	if err != nil {
		return nil, err
	}
	universe, err := source.ResolveUniverse(req)
	if err != nil {
		return nil, err
	}
	if certified != nil {
		if err := structure.VerifyProviderArtifactContext(
			certified, universe, selected,
		); err != nil {
			return nil, err
		}
	}
	plan, err := sourceplan.Build(
		universe,
		selected,
		certifiedInput(req, certified),
	)
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifyProviderSelection(
		universe, plan, certified,
	); err != nil {
		return nil, err
	}
	hydration, err := source.NewHydrationRequest(
		plan.LocalFileIDs(),
		plan.LocalSyntheticPackages(),
	)
	if err != nil {
		return nil, err
	}
	if err := source.HydrateUniverse(universe, hydration); err != nil {
		return nil, err
	}
	graph, index, err := structure.BuildPlanned(universe, plan, certified)
	if err != nil {
		return nil, err
	}
	facts, err := selectionfacts.Materialize(
		universe, graph, index, plan, selected, certified,
	)
	if err != nil {
		return nil, err
	}
	selections, err := scope.SelectDefinitions(
		universe, graph, facts, selected,
	)
	if err != nil {
		return nil, err
	}
	executableInventory, err := executable.Build(graph, index, selections)
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifyStage1(
		req,
		universe,
		plan,
		graph,
		facts,
		selections,
		executableInventory,
		certified,
	); err != nil {
		return nil, err
	}
	hydrationStats := universe.HydrationStats()
	workspace, err := source.Finalize(universe)
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifySourceUniverse(workspace, req); err != nil {
		return nil, err
	}
	if err := stagecheck.VerifyFinalizedStage1(
		universe, workspace, graph, selections, executableInventory,
	); err != nil {
		return nil, err
	}
	return &Inspection{
		workspace: workspace, plan: plan, graph: graph, facts: facts,
		selections: selections, executable: executableInventory,
		hydration: hydrationStats,
	}, nil
}

// AuditCatalog derives and independently certifies provider structural graphs
// from local syntax. It is the only provider-artifact production route.
func AuditCatalog(
	req source.Request,
	output string,
) (structure.ProviderWriteResult, error) {
	selected, err := contract.Resolve(
		req.ProviderContract,
		req.ProviderContractDigest,
		req.ProviderContractArtifact,
	)
	if err != nil {
		return structure.ProviderWriteResult{}, err
	}
	universe, err := source.ResolveUniverse(req)
	if err != nil {
		return structure.ProviderWriteResult{}, err
	}
	plan, err := sourceplan.BuildForAudit(universe, selected)
	if err != nil {
		return structure.ProviderWriteResult{}, err
	}
	packageIDs, err := providerPackageIDs(universe, plan)
	if err != nil {
		return structure.ProviderWriteResult{}, err
	}
	writer, err := structure.NewProviderArtifactWriter(
		universe, selected, output,
	)
	if err != nil {
		return structure.ProviderWriteResult{}, err
	}
	defer writer.Abort()
	for _, packageID := range packageIDs {
		shard, err := auditProviderPackage(
			req, universe, selected, plan, packageID,
		)
		if err != nil {
			return structure.ProviderWriteResult{}, err
		}
		if err := writer.Append(shard); err != nil {
			return structure.ProviderWriteResult{}, err
		}
	}
	manifest, err := writer.ManifestArtifact()
	if err != nil {
		return structure.ProviderWriteResult{}, err
	}
	if err := stagecheck.VerifyProviderManifest(
		universe, selected, plan, manifest,
	); err != nil {
		return structure.ProviderWriteResult{}, err
	}
	workspace, err := source.FinalizeResolved(universe)
	if err != nil {
		return structure.ProviderWriteResult{}, err
	}
	if err := stagecheck.VerifySourceUniverse(workspace, req); err != nil {
		return structure.ProviderWriteResult{}, err
	}
	return writer.Finish()
}

// AuditVerify independently re-derives the provider graph and exact-joins a
// stored artifact. The artifact seal is integrity evidence; certification is
// this independent derivation.
func AuditVerify(req source.Request, path string) error {
	stored, err := structure.DecodeProviderArtifact(path, "")
	if err != nil {
		return err
	}
	selected, err := contract.Resolve(
		req.ProviderContract,
		req.ProviderContractDigest,
		req.ProviderContractArtifact,
	)
	if err != nil {
		return err
	}
	universe, err := source.ResolveUniverse(req)
	if err != nil {
		return err
	}
	if err := structure.VerifyProviderArtifactContext(
		stored, universe, selected,
	); err != nil {
		return err
	}
	plan, err := sourceplan.BuildForAudit(universe, selected)
	if err != nil {
		return err
	}
	if err := stagecheck.VerifyProviderManifest(
		universe, selected, plan, stored,
	); err != nil {
		return err
	}
	packageIDs, err := providerPackageIDs(universe, plan)
	if err != nil {
		return err
	}
	for _, packageID := range packageIDs {
		if err := verifyProviderPackage(
			req,
			universe,
			selected,
			plan,
			packageID,
			stored,
		); err != nil {
			return err
		}
	}
	workspace, err := source.FinalizeResolved(universe)
	if err != nil {
		return err
	}
	if err := stagecheck.VerifySourceUniverse(workspace, req); err != nil {
		return err
	}
	return nil
}

func auditProviderPackage(
	req source.Request,
	base *source.Universe,
	selected contract.Contract,
	plan *sourceplan.Plan,
	packageID identity.PackageID,
) (*structure.ProviderArtifact, error) {
	derived, err := deriveProviderPackage(
		req, base, selected, packageID,
	)
	if err != nil {
		return nil, err
	}
	defer derived.discard()
	artifact, err := structure.ProduceProviderPackageArtifact(
		derived.universe,
		selected,
		plan,
		packageID,
		derived.graph,
		derived.facts.CertifiedFacts(),
	)
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifyProducedProviderPackageArtifact(
		derived.universe,
		selected,
		plan,
		packageID,
		derived.graph,
		derived.facts,
		artifact,
	); err != nil {
		return nil, err
	}
	return artifact, nil
}

func verifyProviderPackage(
	req source.Request,
	base *source.Universe,
	selected contract.Contract,
	plan *sourceplan.Plan,
	packageID identity.PackageID,
	stored *structure.ProviderArtifact,
) error {
	derived, err := deriveProviderPackage(
		req, base, selected, packageID,
	)
	if err != nil {
		return err
	}
	defer derived.discard()
	return stagecheck.VerifyProducedProviderPackageArtifact(
		derived.universe,
		selected,
		plan,
		packageID,
		derived.graph,
		derived.facts,
		stored,
	)
}

type providerPackageDerivation struct {
	universe *source.Universe
	graph    *structure.Graph
	facts    *selectionfacts.Artifact
}

func deriveProviderPackage(
	req source.Request,
	base *source.Universe,
	selected contract.Contract,
	packageID identity.PackageID,
) (_ *providerPackageDerivation, resultErr error) {
	fork, err := source.ForkForHydration(
		base, []identity.PackageID{packageID},
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resultErr != nil {
			source.DiscardHydratedUniverse(fork)
		}
	}()
	files, synthetic, err := packageHydration(fork, packageID)
	if err != nil {
		return nil, err
	}
	hydration, err := source.NewHydrationRequest(files, synthetic)
	if err != nil {
		return nil, err
	}
	if err := source.HydrateUniverse(fork, hydration); err != nil {
		return nil, err
	}
	graph, index, err := structure.BuildPackages(
		fork, []identity.PackageID{packageID},
	)
	if err != nil {
		return nil, err
	}
	facts, err := selectionfacts.MaterializeForAudit(
		fork, graph, index, selected,
	)
	if err != nil {
		return nil, err
	}
	selections, err := scope.SelectDefinitions(
		fork, graph, facts, selected,
	)
	if err != nil {
		return nil, err
	}
	executableInventory, err := executable.Build(
		graph, index, selections,
	)
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifyExtractedPackageStage1(
		req,
		fork,
		packageID,
		graph,
		facts,
		selections,
		executableInventory,
	); err != nil {
		return nil, err
	}
	return &providerPackageDerivation{
		universe: fork, graph: graph, facts: facts,
	}, nil
}

func (d *providerPackageDerivation) discard() {
	source.DiscardHydratedUniverse(d.universe)
}

func providerPackageIDs(
	universe *source.Universe,
	plan *sourceplan.Plan,
) ([]identity.PackageID, error) {
	filePackages := map[identity.FileID]identity.PackageID{}
	for _, pkg := range universe.Packages() {
		for _, file := range pkg.Files() {
			filePackages[file.ID()] = pkg.ID()
		}
	}
	set := map[identity.PackageID]bool{}
	for _, decision := range plan.Files() {
		if decision.Kind() != sourceplan.KindCertifiedGraph {
			continue
		}
		packageID := filePackages[decision.ID()]
		if packageID.IsZero() {
			return nil, fmt.Errorf(
				"provider plan file has no package %s", decision.ID(),
			)
		}
		set[packageID] = true
	}
	for _, decision := range plan.SyntheticOwners() {
		if decision.Kind() == sourceplan.KindCertifiedGraph {
			set[decision.Package()] = true
		}
	}
	out := make([]identity.PackageID, 0, len(set))
	for packageID := range set {
		out = append(out, packageID)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].String() < out[j].String()
	})
	return out, nil
}

func packageHydration(
	universe *source.Universe,
	packageID identity.PackageID,
) ([]identity.FileID, []identity.PackageID, error) {
	for _, pkg := range universe.Packages() {
		if pkg.ID() != packageID {
			continue
		}
		files := make([]identity.FileID, 0, len(pkg.Files()))
		for _, file := range pkg.Files() {
			files = append(files, file.ID())
		}
		var synthetic []identity.PackageID
		if pkg.HasCheckedView() {
			synthetic = append(synthetic, packageID)
		}
		return files, synthetic, nil
	}
	return nil, nil, fmt.Errorf(
		"provider package %s is absent from source", packageID,
	)
}

func decodeSelectedArtifact(
	req source.Request,
) (*structure.ProviderArtifact, error) {
	if req.AuditArtifact == "" {
		if req.AuditArtifactDigest != "" {
			return nil, fmt.Errorf(
				"provider artifact digest is present without an artifact",
			)
		}
		return nil, nil
	}
	if req.AuditArtifactDigest == "" {
		return nil, fmt.Errorf(
			"provider artifact requires an externally selected file digest",
		)
	}
	return structure.DecodeProviderArtifact(
		req.AuditArtifact, req.AuditArtifactDigest,
	)
}

func certifiedInput(
	req source.Request,
	artifact *structure.ProviderArtifact,
) sourceplan.CertifiedInput {
	if artifact == nil {
		return sourceplan.CertifiedInput{}
	}
	return sourceplan.CertifiedInput{
		Digest:   req.AuditArtifactDigest,
		Files:    artifact.FileIDs(),
		Packages: artifact.PackageIDs(),
	}
}
