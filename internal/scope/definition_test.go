package scope

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestDefinitionKindDepthCompatibilityMatrix(t *testing.T) {
	definitions := compatibilityDefinitions(t)
	allowed := map[string]map[source.LanguageDisposition][]contract.EvidenceDepth{
		"func-decl": {
			source.DispositionOrdinarySource: {
				contract.DepthFullSemantic,
				contract.DepthDeclarationContract,
				contract.DepthExternalBoundary,
			},
			source.DispositionBuiltinUniverse: {contract.DepthIntrinsic},
			source.DispositionUnsafeIntrinsic: {contract.DepthIntrinsic},
		},
		"func-lit": {
			source.DispositionOrdinarySource: {
				contract.DepthFullSemantic,
				contract.DepthDeclarationContract,
				contract.DepthExternalBoundary,
			},
			source.DispositionBuiltinUniverse: {contract.DepthIntrinsic},
			source.DispositionUnsafeIntrinsic: {contract.DepthIntrinsic},
		},
		"package-initializer": {
			source.DispositionOrdinarySource: {
				contract.DepthFullSemantic,
				contract.DepthDeclarationContract,
				contract.DepthExternalBoundary,
			},
			source.DispositionBuiltinUniverse: {contract.DepthIntrinsic},
			source.DispositionUnsafeIntrinsic: {contract.DepthIntrinsic},
		},
		"bodyless": {
			source.DispositionOrdinarySource: {
				contract.DepthDeclarationContract,
				contract.DepthExternalBoundary,
			},
			source.DispositionBuiltinUniverse: {contract.DepthIntrinsic},
			source.DispositionUnsafeIntrinsic: {contract.DepthIntrinsic},
		},
		"implicit": {
			source.DispositionOrdinarySource: {
				contract.DepthFullSemantic,
				contract.DepthDeclarationContract,
				contract.DepthExternalBoundary,
			},
			source.DispositionBuiltinUniverse: {contract.DepthIntrinsic},
			source.DispositionUnsafeIntrinsic: {contract.DepthIntrinsic},
		},
		"synthetic": {
			source.DispositionOrdinarySource: {
				contract.DepthExternalBoundary,
			},
		},
	}
	depths := []contract.EvidenceDepth{
		contract.DepthInvalid,
		contract.DepthFullSemantic,
		contract.DepthDeclarationContract,
		contract.DepthExternalBoundary,
		contract.DepthIntrinsic,
	}
	dispositions := []source.LanguageDisposition{
		source.LanguageDispositionInvalid,
		source.DispositionOrdinarySource,
		source.DispositionBuiltinUniverse,
		source.DispositionUnsafeIntrinsic,
	}
	represented := map[identity.DefinitionKind]bool{}
	for _, definition := range definitions {
		represented[definition.id.Kind()] = true
		for _, disposition := range dispositions {
			for _, depth := range depths {
				want := containsDepth(
					allowed[definition.name][disposition],
					depth,
				)
				err := validateCompatibility(
					definition.id,
					depth,
					disposition,
				)
				if (err == nil) != want {
					t.Errorf(
						"%s/%s/%s error=%v, valid=%t",
						definition.name,
						disposition,
						depth,
						err,
						want,
					)
				}
			}
		}
	}
	for kind := identity.DefinitionKind(1); kind.Valid(); kind++ {
		if !represented[kind] {
			t.Errorf("compatibility matrix omits definition kind %s", kind)
		}
	}
}

type compatibilityDefinition struct {
	name string
	id   identity.DefinitionID
}

func compatibilityDefinitions(
	t *testing.T,
) []compatibilityDefinition {
	t.Helper()
	module, err := identity.NewModuleID(
		"example.com/compatibility",
		"v1.0.0",
	)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := identity.NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := identity.NewPackageID(
		owner,
		"example.com/compatibility",
	)
	if err != nil {
		t.Fatal(err)
	}
	file, err := identity.NewFileID(owner, "compatibility.go")
	if err != nil {
		t.Fatal(err)
	}
	span, err := identity.NewSpanID(file, 10, 20)
	if err != nil {
		t.Fatal(err)
	}
	root, err := identity.NewOccurrenceID(span, 47)
	if err != nil {
		t.Fatal(err)
	}
	var definitions []compatibilityDefinition
	for _, record := range []struct {
		name string
		kind identity.DefinitionKind
	}{
		{"func-decl", identity.DefinitionFuncDecl},
		{"func-lit", identity.DefinitionFuncLit},
		{"package-initializer", identity.DefinitionPackageInitializer},
		{"bodyless", identity.DefinitionBodylessDecl},
	} {
		definition, err := identity.NewSourceDefinitionID(root, record.kind)
		if err != nil {
			t.Fatal(err)
		}
		definitions = append(definitions, compatibilityDefinition{
			name: record.name,
			id:   definition,
		})
	}
	implicitDefinition, err := identity.NewImplicitDefinitionID(
		pkg,
		identity.ImplicitDefinitionPackageInit,
	)
	if err != nil {
		t.Fatal(err)
	}
	syntheticDefinition, err := identity.NewSyntheticDefinitionID(
		pkg,
		identity.SyntheticDefinitionAdapter,
		"adapter",
	)
	if err != nil {
		t.Fatal(err)
	}
	definitions = append(
		definitions,
		compatibilityDefinition{
			name: "implicit",
			id:   implicitDefinition,
		},
		compatibilityDefinition{
			name: "synthetic",
			id:   syntheticDefinition,
		},
	)
	return definitions
}

func containsDepth(
	depths []contract.EvidenceDepth,
	target contract.EvidenceDepth,
) bool {
	for _, depth := range depths {
		if depth == target {
			return true
		}
	}
	return false
}
