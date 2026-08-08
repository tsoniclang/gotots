package artifact

import (
	"bytes"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/callableabi"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestCallableABIIsOwnedBySignatureFacet(t *testing.T) {
	packageType := types.NewPackage("example.test/abi", "abi")
	function := types.NewFunc(
		token.Pos(1),
		packageType,
		"Read",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	consumer := types.NewFunc(
		token.Pos(2),
		packageType,
		"Use",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	providerOwner := api.MustSourceArtifactOwner(function)
	consumerOwner := api.MustSourceArtifactOwner(consumer)
	graph := NewGraph(compareArtifactTestObjects)
	initial := callableABITestContract(t, "Pointer<number> | undefined")
	if err := graph.AdmitCallableABI(providerOwner, initial); err != nil {
		t.Fatal(err)
	}
	providerContract := callableABIFacetContract(
		t,
		api.ArtifactFacetCallableSignature,
		"function Read(value: number): number",
	)
	if err := graph.Commit(providerOwner, providerContract, nil); err != nil {
		t.Fatal(err)
	}
	dependency, err := api.NewArtifactDependency(
		providerOwner,
		api.ArtifactFacetCallableSignature,
	)
	if err != nil {
		t.Fatal(err)
	}
	consumerContract := callableABIFacetContract(
		t,
		api.ArtifactFacetImplementation,
		"Read(value)",
	)
	if err := graph.Commit(
		consumerOwner,
		consumerContract,
		[]api.ArtifactDependency{dependency},
	); err != nil {
		t.Fatal(err)
	}
	before := graph.records[providerOwner].contract
	if err := graph.Commit(providerOwner, providerContract, nil); err != nil {
		t.Fatal(err)
	}
	if graph.HasPending() {
		t.Fatal("identical callable signature dirtied its consumer")
	}
	revised := callableABITestContract(t, "Pointer<bigint> | undefined")
	if err := graph.AdmitCallableABI(providerOwner, revised); err != nil {
		t.Fatal(err)
	}
	after := graph.records[providerOwner].contract
	if bytes.Equal(
		artifactFacetBytes(before, api.ArtifactFacetCallableSignature),
		artifactFacetBytes(after, api.ArtifactFacetCallableSignature),
	) {
		t.Fatal("target-type revision did not change the callable-signature facet")
	}
	dirty, ok := graph.NextDirty()
	if !ok || dirty != consumerOwner {
		t.Fatalf("dirty consumer = %v, %t; want Use", dirty, ok)
	}
	selected, ok := graph.CallableABI(providerOwner)
	if !ok || selected.Fingerprint() != revised.Fingerprint() {
		t.Fatal("graph did not retain the revised canonical callable ABI")
	}
}

func callableABIFacetContract(
	t *testing.T,
	facet api.ArtifactFacet,
	value string,
) Contract {
	t.Helper()
	contract, err := NewContractFacet(facet, []byte(value))
	if err != nil {
		t.Fatal(err)
	}
	return contract
}

func callableABITestContract(
	t *testing.T,
	targetType string,
) callableabi.Callable {
	t.Helper()
	parameter, err := callableabi.NewParameter(targetType)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := callableabi.PackageFunctionIdentity(
		"example.test/abi",
		"Read",
	)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := callableabi.New(
		identity,
		[]callableabi.Parameter{parameter},
		"number",
	)
	if err != nil {
		t.Fatal(err)
	}
	return selected
}
