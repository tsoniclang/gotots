// Package compiler owns pipeline sequencing only. Semantic and structural
// decisions remain in their phase owners, and every produced artifact passes
// its blocking independent verifier before downstream consumption.
package compiler

import (
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

// Inspection is the immutable verified Stage-1 result.
type Inspection struct {
	workspace    *source.Workspace
	plan         *sourceplan.Plan
	graph        *structure.Graph
	facts        *selectionfacts.Artifact
	selections   *scope.DefinitionSelections
	executable   *executable.Inventory
	semantic     *semantic.Model
	semanticWork frontend.Work
	hydration    source.HydrationStats
}

func (i *Inspection) Workspace() *source.Workspace             { return i.workspace }
func (i *Inspection) SourcePlan() *sourceplan.Plan             { return i.plan }
func (i *Inspection) Structure() *structure.Graph              { return i.graph }
func (i *Inspection) SelectionFacts() *selectionfacts.Artifact { return i.facts }
func (i *Inspection) Selections() *scope.DefinitionSelections  { return i.selections }
func (i *Inspection) Executable() *executable.Inventory        { return i.executable }
func (i *Inspection) Semantic() *semantic.Model                { return i.semantic }
func (i *Inspection) SemanticWork() frontend.Work              { return i.semanticWork }
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
	certifiedStructure, err := decodeSelectedStructureArtifact(req)
	if err != nil {
		return nil, err
	}
	certifiedSemantic, err := decodeSelectedSemanticArtifact(req)
	if err != nil {
		return nil, err
	}
	universe, err := source.ResolveUniverse(req)
	if err != nil {
		return nil, err
	}
	if certifiedStructure != nil {
		if err := structure.VerifyProviderArtifactContext(
			certifiedStructure, universe, selected,
		); err != nil {
			return nil, err
		}
	}
	if certifiedSemantic != nil {
		if err := certifiedSemantic.VerifyContext(
			semanticProviderContext(universe, selected),
			req.ProviderStructureDigest,
		); err != nil {
			return nil, err
		}
	}
	plan, err := sourceplan.Build(
		universe,
		selected,
		certifiedInput(req, certifiedStructure),
	)
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifyProviderSelection(
		universe, plan, certifiedStructure,
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
	graph, index, err := structure.BuildPlanned(
		universe, plan, certifiedStructure,
	)
	if err != nil {
		return nil, err
	}
	facts, err := selectionfacts.Materialize(
		universe, graph, index, plan, selected, certifiedStructure,
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
		certifiedStructure,
	); err != nil {
		return nil, err
	}
	semanticResult, err := frontend.Materialize(
		universe,
		graph,
		index,
		facts,
		selections,
		executableInventory,
		plan,
		certifiedSemantic,
	)
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifyStage2(
		universe,
		plan,
		graph,
		index,
		facts,
		selections,
		executableInventory,
		semanticResult.Model(),
		certifiedSemantic,
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
	if err := stagecheck.VerifyFinalizedStage2(
		semanticResult.Model(),
	); err != nil {
		return nil, err
	}
	return &Inspection{
		workspace: workspace, plan: plan, graph: graph, facts: facts,
		selections: selections, executable: executableInventory,
		semantic:     semanticResult.Model(),
		semanticWork: semanticResult.Work(),
		hydration:    hydrationStats,
	}, nil
}
