package compiler

import (
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestMixedLocalAndCertifiedFilesUseOneAuthorityEach(t *testing.T) {
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/mixed-authority\n\ngo 1.26.0\n",
	)
	writeCompilerFile(
		t,
		directory,
		"automatic.go",
		"package mixed\n\nfunc Automatic() int { return 1 }\n",
	)
	writeCompilerFile(
		t,
		directory,
		"provided.go",
		"package mixed\n\nfunc Provided() int { return 2 }\n",
	)
	base, err := InspectConstructs(source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	baseRecords := collectStage1Definitions(t, base)
	automaticDefinitions := findDefinitionsByName(
		baseRecords,
		"Automatic",
	)
	providedDefinitions := findDefinitionsByName(
		baseRecords,
		"Provided",
	)
	if len(automaticDefinitions) != 1 ||
		len(providedDefinitions) != 1 {
		t.Fatalf(
			"base Automatic/Provided definitions=%d/%d",
			len(automaticDefinitions),
			len(providedDefinitions),
		)
	}
	automatic := automaticDefinitions[0]
	provided := providedDefinitions[0]
	contractPath := writeDepthContract(
		t,
		"mixed-authority@v1",
		automatic,
	)
	request := source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract:         "mixed-authority@v1",
		ProviderContractArtifact: contractPath,
	}
	output := t.TempDir()
	structurePath := filepath.Join(output, "provider.structure.gotots")
	semanticPath := filepath.Join(output, "provider.semantic.gotots")
	provider, err := AuditCatalog(request, structurePath, semanticPath)
	if err != nil {
		t.Fatal(err)
	}
	request.ProviderStructureArtifact = structurePath
	request.ProviderStructureDigest = provider.Structure.Digest
	request.ProviderSemanticArtifact = semanticPath
	request.ProviderSemanticDigest = provider.Semantic.Digest
	inspection, err := InspectConstructs(request)
	if err != nil {
		t.Fatal(err)
	}

	automaticPlan, automaticPresent :=
		inspection.SourcePlan().For(automatic.File())
	providedPlan, providedPresent :=
		inspection.SourcePlan().For(provided.File())
	if !automaticPresent ||
		automaticPlan.Kind() != sourceplan.KindLocalSyntax ||
		!providedPresent ||
		providedPlan.Kind() != sourceplan.KindCertifiedGraph {
		t.Fatalf(
			"file authorities Automatic=%s/%t Provided=%s/%t",
			automaticPlan.Kind(),
			automaticPresent,
			providedPlan.Kind(),
			providedPresent,
		)
	}
	records := collectStage1Definitions(t, inspection)
	if len(findDefinitionsByName(records, "Automatic")) != 1 ||
		len(findDefinitionsByName(records, "Provided")) != 1 {
		t.Fatal("mixed local/certified projection lost or duplicated a definition")
	}
	if _, present := inspection.Structure().ResidentDefinition(
		automatic,
	); !present {
		t.Fatal("automatic definition is not resident")
	}
	if _, present := inspection.Structure().ResidentDefinition(
		provided,
	); present {
		t.Fatal("certified definition was duplicated into resident detail")
	}
	automaticSelection, automaticSelected :=
		inspection.Selections().For(automatic)
	providedSelection, providedSelected :=
		inspection.Selections().For(provided)
	if !automaticSelected ||
		automaticSelection.Depth() != contract.DepthFullSemantic ||
		!providedSelected ||
		providedSelection.Depth() != contract.DepthDeclarationContract {
		t.Fatalf(
			"definition depths Automatic=%s/%t Provided=%s/%t",
			automaticSelection.Depth(),
			automaticSelected,
			providedSelection.Depth(),
			providedSelected,
		)
	}
	if _, present := inspection.Executable().For(automatic); !present {
		t.Fatal("automatic definition has no executable region")
	}
	if _, present := inspection.Executable().For(provided); present {
		t.Fatal("certified declaration owns executable occurrences")
	}
	manifest := inspection.Structure().ProviderManifestStats()
	projection := inspection.Structure().ProviderProjectionStats()
	if manifest.PackageContexts != 1 ||
		manifest.Files != 1 ||
		projection.ShardLoads != 1 ||
		projection.MaxResidentPackages != 1 {
		t.Fatalf(
			"mixed provider manifest/projection=%+v/%+v",
			manifest,
			projection,
		)
	}
}
