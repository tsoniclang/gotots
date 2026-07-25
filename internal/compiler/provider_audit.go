package compiler

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/frontend"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
	"github.com/tsoniclang/gotots/internal/stagecheck"
)

// ProviderWriteResult reports the separately encoded structural and semantic
// provider authorities produced by one bounded package-at-a-time derivation.
type ProviderWriteResult struct {
	Structure    structure.ProviderWriteResult
	Semantic     semantic.ProviderWriteResult
	SemanticWork frontend.Work
}

// AuditCatalog derives provider structural and semantic artifacts from the
// same package-at-a-time transient checker graph.
func AuditCatalog(
	req source.Request,
	structureOutput string,
	semanticOutput string,
) (ProviderWriteResult, error) {
	selected, err := contract.Resolve(
		req.ProviderContract,
		req.ProviderContractDigest,
		req.ProviderContractArtifact,
	)
	if err != nil {
		return ProviderWriteResult{}, err
	}
	universe, err := source.ResolveUniverse(req)
	if err != nil {
		return ProviderWriteResult{}, err
	}
	plan, err := sourceplan.BuildForAudit(universe, selected)
	if err != nil {
		return ProviderWriteResult{}, err
	}
	packageIDs, err := providerPackageIDs(universe, plan)
	if err != nil {
		return ProviderWriteResult{}, err
	}
	structureWriter, err := structure.NewProviderArtifactWriter(
		universe, selected, structureOutput,
	)
	if err != nil {
		return ProviderWriteResult{}, err
	}
	defer structureWriter.Abort()
	semanticWriter, err := semantic.NewProviderArtifactWriter(
		semanticProviderContext(universe, selected),
		semanticOutput,
	)
	if err != nil {
		return ProviderWriteResult{}, err
	}
	defer semanticWriter.Abort()
	semanticWork := frontend.Work{}
	for _, packageID := range packageIDs {
		derived, err := deriveProviderPackage(
			req, universe, selected, packageID,
		)
		if err != nil {
			return ProviderWriteResult{}, err
		}
		structureShard, semanticPackage, packageWork, err :=
			auditProviderPackage(
				selected, plan, packageID, derived,
			)
		derived.discard()
		if err != nil {
			return ProviderWriteResult{}, err
		}
		if err := structureWriter.Append(structureShard); err != nil {
			return ProviderWriteResult{}, err
		}
		if err := semanticWriter.Append(semanticPackage); err != nil {
			return ProviderWriteResult{}, err
		}
		semanticWork = semanticWork.Plus(packageWork)
	}
	manifest, err := structureWriter.ManifestArtifact()
	if err != nil {
		return ProviderWriteResult{}, err
	}
	if err := stagecheck.VerifyProviderManifest(
		universe, selected, plan, manifest,
	); err != nil {
		return ProviderWriteResult{}, err
	}
	workspace, err := source.FinalizeResolved(universe)
	if err != nil {
		return ProviderWriteResult{}, err
	}
	if err := stagecheck.VerifySourceUniverse(workspace, req); err != nil {
		return ProviderWriteResult{}, err
	}
	structureResult, err := structureWriter.Finish()
	if err != nil {
		return ProviderWriteResult{}, err
	}
	semanticResult, err := semanticWriter.Finish(
		structureResult.Digest,
	)
	if err != nil {
		return ProviderWriteResult{}, err
	}
	return ProviderWriteResult{
		Structure:    structureResult,
		Semantic:     semanticResult,
		SemanticWork: semanticWork,
	}, nil
}

// AuditVerify independently re-derives the provider graph and exact-joins a
// stored artifact. The artifact seal is integrity evidence; certification is
// this independent derivation.
func AuditVerify(
	req source.Request,
	structurePath string,
	semanticPath string,
) error {
	storedStructure, err := structure.DecodeProviderArtifact(
		structurePath, "",
	)
	if err != nil {
		return err
	}
	storedSemantic, err := semantic.DecodeProviderArtifact(
		semanticPath, "",
	)
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
		storedStructure, universe, selected,
	); err != nil {
		return err
	}
	if err := storedSemantic.VerifyContext(
		semanticProviderContext(universe, selected),
		storedStructure.Digest(),
	); err != nil {
		return err
	}
	plan, err := sourceplan.BuildForAudit(universe, selected)
	if err != nil {
		return err
	}
	if err := stagecheck.VerifyProviderManifest(
		universe, selected, plan, storedStructure,
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
			storedStructure,
			storedSemantic,
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
	selected contract.Contract,
	plan *sourceplan.Plan,
	packageID identity.PackageID,
	derived *providerPackageDerivation,
) (
	*structure.ProviderArtifact,
	semantic.Package,
	frontend.Work,
	error,
) {
	artifact, err := structure.ProduceProviderPackageArtifact(
		derived.universe,
		selected,
		plan,
		packageID,
		derived.graph,
		derived.facts.CertifiedFacts(),
	)
	if err != nil {
		return nil, semantic.Package{}, frontend.Work{}, err
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
		return nil, semantic.Package{}, frontend.Work{}, err
	}
	semanticPackage, work, err := frontend.MaterializeProviderPackage(
		derived.universe,
		derived.graph,
		derived.index,
		derived.facts,
		derived.selections,
		derived.executable,
		plan,
	)
	if err != nil {
		return nil, semantic.Package{}, frontend.Work{}, err
	}
	return artifact, semanticPackage, work, nil
}

func verifyProviderPackage(
	req source.Request,
	base *source.Universe,
	selected contract.Contract,
	plan *sourceplan.Plan,
	packageID identity.PackageID,
	storedStructure *structure.ProviderArtifact,
	storedSemantic *semantic.ProviderArtifact,
) error {
	derived, err := deriveProviderPackage(
		req, base, selected, packageID,
	)
	if err != nil {
		return err
	}
	defer derived.discard()
	if err := stagecheck.VerifyProducedProviderPackageArtifact(
		derived.universe,
		selected,
		plan,
		packageID,
		derived.graph,
		derived.facts,
		storedStructure,
	); err != nil {
		return err
	}
	return stagecheck.VerifyProducedSemanticPackage(
		derived.universe,
		plan,
		packageID,
		derived.graph,
		derived.index,
		derived.facts,
		derived.selections,
		derived.executable,
		storedSemantic,
	)
}

type providerPackageDerivation struct {
	universe   *source.Universe
	graph      *structure.Graph
	index      *structure.TransientIndex
	facts      *selectionfacts.Artifact
	selections *scope.DefinitionSelections
	executable *executable.Inventory
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
	if err := index.SealForStage2(); err != nil {
		return nil, err
	}
	return &providerPackageDerivation{
		universe:   fork,
		graph:      graph,
		index:      index,
		facts:      facts,
		selections: selections,
		executable: executableInventory,
	}, nil
}

func (d *providerPackageDerivation) discard() {
	if d == nil {
		return
	}
	_ = source.DiscardHydratedUniverse(d.universe)
	d.universe = nil
	d.graph = nil
	d.index = nil
	d.facts = nil
	d.selections = nil
	d.executable = nil
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
		return out[i].Compare(out[j]) < 0
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

func decodeSelectedStructureArtifact(
	req source.Request,
) (*structure.ProviderArtifact, error) {
	if req.ProviderStructureArtifact == "" {
		if req.ProviderStructureDigest != "" {
			return nil, fmt.Errorf(
				"provider structure digest is present without an artifact",
			)
		}
		return nil, nil
	}
	if req.ProviderStructureDigest == "" {
		return nil, fmt.Errorf(
			"provider structure requires an externally selected file digest",
		)
	}
	return structure.DecodeProviderArtifact(
		req.ProviderStructureArtifact,
		req.ProviderStructureDigest,
	)
}

func decodeSelectedSemanticArtifact(
	req source.Request,
) (*semantic.ProviderArtifact, error) {
	if req.ProviderSemanticArtifact == "" {
		if req.ProviderSemanticDigest != "" {
			return nil, fmt.Errorf(
				"provider semantic digest is present without an artifact",
			)
		}
		return nil, nil
	}
	if req.ProviderSemanticDigest == "" {
		return nil, fmt.Errorf(
			"provider semantics require an externally selected file digest",
		)
	}
	return semantic.DecodeProviderArtifact(
		req.ProviderSemanticArtifact,
		req.ProviderSemanticDigest,
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
		Digest:   req.ProviderStructureDigest,
		Files:    artifact.FileIDs(),
		Packages: artifact.PackageIDs(),
	}
}

func semanticProviderContext(
	universe *source.Universe,
	selected contract.Contract,
) semantic.ProviderArtifactContext {
	return semantic.ProviderArtifactContext{
		ToolchainDigest: universe.Toolchain().BinaryDigest(),
		ConfigurationDigest: universe.Toolchain().
			BuildConfigurationDigest(),
		ContractID:          selected.ID(),
		ContractFingerprint: selected.Fingerprint(),
	}
}
