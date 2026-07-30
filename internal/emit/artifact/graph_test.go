package artifact

import (
	"errors"
	"go/token"
	"go/types"
	"reflect"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestArtifactGraphSuppressesUnchangedAndIsolatesFacets(t *testing.T) {
	provider := artifactTestObject("Provider", 10)
	callConsumer := artifactTestObject("CallConsumer", 20)
	staticConsumer := artifactTestObject("StaticConsumer", 30)
	graph := NewGraph(compareArtifactTestObjects)

	commitArtifactTestRevision(
		t,
		graph,
		provider,
		artifactTestContract(t, map[api.ArtifactFacet]string{
			api.ArtifactFacetCallableSignature: "call-a",
			api.ArtifactFacetStaticSurface:     "static-a",
		}),
	)
	commitArtifactTestRevision(
		t,
		graph,
		callConsumer,
		artifactCallable("consumer"),
		artifactTestDependency(
			t,
			provider,
			api.ArtifactFacetCallableSignature,
		),
	)
	commitArtifactTestRevision(
		t,
		graph,
		staticConsumer,
		artifactCallable("consumer"),
		artifactTestDependency(
			t,
			provider,
			api.ArtifactFacetStaticSurface,
		),
	)

	commitArtifactTestRevision(
		t,
		graph,
		provider,
		artifactTestContract(t, map[api.ArtifactFacet]string{
			api.ArtifactFacetCallableSignature: "call-a",
			api.ArtifactFacetStaticSurface:     "static-a",
		}),
	)
	if graph.HasPending() {
		t.Fatal("equal contract dirtied a consumer")
	}
	if len(graph.records[artifactTestOwner(provider)].history.entries) != 1 {
		t.Fatal("equal contract added convergence history")
	}

	commitArtifactTestRevision(
		t,
		graph,
		provider,
		artifactTestContract(t, map[api.ArtifactFacet]string{
			api.ArtifactFacetCallableSignature: "call-a",
			api.ArtifactFacetStaticSurface:     "static-b",
		}),
	)
	dirty, ok := graph.NextDirty()
	if !ok || dirty != artifactTestOwner(staticConsumer) {
		t.Fatalf("dirty = %v, %t; want static consumer", dirty, ok)
	}
	if graph.HasPending() {
		t.Fatal("static change dirtied an unrelated callable consumer")
	}
	if len(graph.records[artifactTestOwner(provider)].history.entries) != 2 {
		t.Fatal("changed contract did not add exactly one history entry")
	}
	if graph.FacetRevision(
		artifactTestOwner(provider),
		api.ArtifactFacetCallableSignature,
	) != 1 ||
		graph.FacetRevision(
			artifactTestOwner(provider),
			api.ArtifactFacetStaticSurface,
		) != 2 {
		t.Fatal("facet revisions did not isolate the changed contract")
	}
}

func TestArtifactGraphNeverRoutesByProviderSpelling(t *testing.T) {
	first := artifactTestObject("Provider", 10)
	forged := artifactTestObject("Provider", 11)
	consumer := artifactTestObject("Consumer", 20)
	graph := NewGraph(compareArtifactTestObjects)
	commitArtifactTestRevision(t, graph, first, artifactCallable("first-a"))
	commitArtifactTestRevision(t, graph, forged, artifactCallable("forged-a"))
	commitArtifactTestRevision(
		t,
		graph,
		consumer,
		artifactCallable("consumer"),
		artifactTestDependency(
			t,
			first,
			api.ArtifactFacetCallableSignature,
		),
	)
	commitArtifactTestRevision(t, graph, forged, artifactCallable("forged-b"))
	if graph.HasPending() {
		t.Fatal("same-spelling provider dirtied the exact provider's consumer")
	}
	commitArtifactTestRevision(t, graph, first, artifactCallable("first-b"))
	dirty, ok := graph.NextDirty()
	if !ok || dirty != artifactTestOwner(consumer) {
		t.Fatalf("exact-provider dirty = %v, %t", dirty, ok)
	}
}

func TestArtifactGraphPropagatesTransitivelyInStableOrder(t *testing.T) {
	leaf := artifactTestObject("Leaf", 10)
	middle := artifactTestObject("Middle", 20)
	firstRoot := artifactTestObject("FirstRoot", 30)
	secondRoot := artifactTestObject("SecondRoot", 40)
	graph := NewGraph(compareArtifactTestObjects)

	commitArtifactTestRevision(t, graph, leaf, artifactCallable("leaf-a"))
	commitArtifactTestRevision(
		t,
		graph,
		middle,
		artifactCallable("middle-a"),
		artifactTestDependency(
			t,
			leaf,
			api.ArtifactFacetCallableSignature,
		),
		artifactTestDependency(
			t,
			leaf,
			api.ArtifactFacetCallableSignature,
		),
	)
	for _, root := range []types.Object{secondRoot, firstRoot} {
		commitArtifactTestRevision(
			t,
			graph,
			root,
			artifactCallable(root.Name()),
			artifactTestDependency(
				t,
				middle,
				api.ArtifactFacetCallableSignature,
			),
		)
	}
	if graph.edgeCount() != 3 {
		t.Fatalf("edges = %d, want three deduplicated edges", graph.edgeCount())
	}

	commitArtifactTestRevision(t, graph, leaf, artifactCallable("leaf-b"))
	dirty, ok := graph.NextDirty()
	if !ok || dirty != artifactTestOwner(middle) {
		t.Fatalf("first dirty = %v, %t; want middle", dirty, ok)
	}
	commitArtifactTestRevision(
		t,
		graph,
		middle,
		artifactCallable("middle-b"),
		artifactTestDependency(
			t,
			leaf,
			api.ArtifactFacetCallableSignature,
		),
	)
	var order []api.ArtifactOwner
	for {
		dirty, ok := graph.NextDirty()
		if !ok {
			break
		}
		order = append(order, dirty)
	}
	if !reflect.DeepEqual(order, []api.ArtifactOwner{
		artifactTestOwner(firstRoot),
		artifactTestOwner(secondRoot),
	}) {
		t.Fatalf("dirty order = %v", artifactTestNames(order))
	}
}

func TestArtifactGraphReplacesDependenciesTransactionally(t *testing.T) {
	provider := artifactTestObject("Provider", 10)
	consumer := artifactTestObject("Consumer", 20)
	graph := NewGraph(compareArtifactTestObjects)
	commitArtifactTestRevision(t, graph, provider, artifactCallable("a"))
	commitArtifactTestRevision(
		t,
		graph,
		consumer,
		artifactCallable("consumer"),
		artifactTestDependency(
			t,
			provider,
			api.ArtifactFacetCallableSignature,
		),
	)
	commitArtifactTestRevision(
		t,
		graph,
		consumer,
		artifactCallable("consumer"),
	)
	if graph.edgeCount() != 0 {
		t.Fatalf("edges = %d after dependency replacement", graph.edgeCount())
	}
	commitArtifactTestRevision(t, graph, provider, artifactCallable("b"))
	if graph.HasPending() {
		t.Fatal("removed dependency still dirtied its former consumer")
	}
}

func TestArtifactGraphRebuildsConsumerWithoutOutgoingFacets(t *testing.T) {
	provider := artifactTestObject("Provider", 10)
	bodyOnlyConsumer := artifactTestObject("PackageInitialization", 20)
	graph := NewGraph(compareArtifactTestObjects)
	commitArtifactTestRevision(t, graph, provider, artifactCallable("a"))
	commitArtifactTestRevision(
		t,
		graph,
		bodyOnlyConsumer,
		NewContract(),
		artifactTestDependency(
			t,
			provider,
			api.ArtifactFacetCallableSignature,
		),
	)

	commitArtifactTestRevision(t, graph, provider, artifactCallable("b"))
	dirty, ok := graph.NextDirty()
	if !ok || dirty != artifactTestOwner(bodyOnlyConsumer) {
		t.Fatalf("body-only dirty = %v, %t; want consumer", dirty, ok)
	}
	commitArtifactTestRevision(
		t,
		graph,
		bodyOnlyConsumer,
		NewContract(),
		artifactTestDependency(
			t,
			provider,
			api.ArtifactFacetCallableSignature,
		),
	)
	if graph.HasPending() {
		t.Fatal("body-only consumer propagated without an outgoing facet")
	}
}

func TestArtifactGraphOwnsCanonicalContractBytes(t *testing.T) {
	provider := artifactTestObject("Provider", 10)
	consumer := artifactTestObject("Consumer", 20)
	graph := NewGraph(compareArtifactTestObjects)
	encoded := []byte("call-a")
	commitArtifactTestRevision(
		t,
		graph,
		provider,
		artifactTestContractBytes(
			t,
			api.ArtifactFacetCallableSignature,
			encoded,
		),
	)
	commitArtifactTestRevision(
		t,
		graph,
		consumer,
		artifactCallable("consumer"),
		artifactTestDependency(
			t,
			provider,
			api.ArtifactFacetCallableSignature,
		),
	)
	encoded[0] = 'X'
	commitArtifactTestRevision(t, graph, provider, artifactCallable("call-a"))
	if graph.HasPending() {
		t.Fatal("caller-owned contract bytes mutated the graph authority")
	}
}

func TestArtifactGraphRejectsAbsentContract(t *testing.T) {
	graph := NewGraph(compareArtifactTestObjects)
	owner := artifactTestObject("Owner", 10)
	var graphError *GraphError
	if err := graph.Commit(
		artifactTestOwner(owner),
		Contract{},
		nil,
	); !errors.As(err, &graphError) {
		t.Fatalf("absent-contract error = %#v", err)
	}
}

func TestArtifactGraphClosureRejectsMissingProvidersAndFacets(t *testing.T) {
	provider := artifactTestObject("Provider", 10)
	consumer := artifactTestObject("Consumer", 20)
	graph := NewGraph(compareArtifactTestObjects)
	commitArtifactTestRevision(
		t,
		graph,
		consumer,
		artifactCallable("consumer"),
		artifactTestDependency(
			t,
			provider,
			api.ArtifactFacetCallableSignature,
		),
	)
	var graphError *GraphError
	if err := graph.VerifyClosure(); !errors.As(err, &graphError) ||
		graphError.Object != artifactTestOwner(consumer) ||
		graphError.Provider != artifactTestOwner(provider) ||
		graphError.Facet != api.ArtifactFacetCallableSignature {
		t.Fatalf("missing-provider error = %#v", err)
	}

	commitArtifactTestRevision(
		t,
		graph,
		provider,
		artifactTestContract(t, map[api.ArtifactFacet]string{
			api.ArtifactFacetStaticSurface: "static",
		}),
	)
	if err := graph.VerifyClosure(); !errors.As(err, &graphError) {
		t.Fatalf("missing-facet error = %#v", err)
	}

	commitArtifactTestRevision(
		t,
		graph,
		provider,
		artifactTestContract(t, map[api.ArtifactFacet]string{
			api.ArtifactFacetCallableSignature: "call",
			api.ArtifactFacetStaticSurface:     "static",
		}),
	)
	if err := graph.VerifyClosure(); err != nil {
		t.Fatal(err)
	}
}

func TestArtifactGraphClosureDiagnosticsUseStableObjectOrder(t *testing.T) {
	firstProvider := artifactTestObject("FirstProvider", 10)
	secondProvider := artifactTestObject("SecondProvider", 20)
	firstConsumer := artifactTestObject("FirstConsumer", 30)
	secondConsumer := artifactTestObject("SecondConsumer", 40)
	graph := NewGraph(compareArtifactTestObjects)
	commitArtifactTestRevision(
		t,
		graph,
		secondConsumer,
		artifactCallable("second"),
		artifactTestDependency(
			t,
			secondProvider,
			api.ArtifactFacetCallableSignature,
		),
	)
	commitArtifactTestRevision(
		t,
		graph,
		firstConsumer,
		artifactCallable("first"),
		artifactTestDependency(
			t,
			firstProvider,
			api.ArtifactFacetCallableSignature,
		),
		artifactTestDependency(
			t,
			secondProvider,
			api.ArtifactFacetCallableSignature,
		),
	)

	var graphError *GraphError
	if err := graph.VerifyClosure(); !errors.As(err, &graphError) ||
		graphError.Object != artifactTestOwner(firstConsumer) ||
		graphError.Provider != artifactTestOwner(firstProvider) {
		t.Fatalf("closure error = %#v, want first consumer/provider", err)
	}
}

func TestArtifactGraphConvergesCyclesAndRejectsOscillation(t *testing.T) {
	first := artifactTestObject("First", 10)
	second := artifactTestObject("Second", 20)
	graph := NewGraph(compareArtifactTestObjects)
	commitArtifactTestRevision(
		t,
		graph,
		first,
		artifactCallable("first-a"),
		artifactTestDependency(
			t,
			second,
			api.ArtifactFacetCallableSignature,
		),
	)
	commitArtifactTestRevision(
		t,
		graph,
		second,
		artifactCallable("second-a"),
		artifactTestDependency(
			t,
			first,
			api.ArtifactFacetCallableSignature,
		),
	)
	commitArtifactTestRevision(t, graph, first, artifactCallable("first-b"))
	dirty, ok := graph.NextDirty()
	if !ok || dirty != artifactTestOwner(second) {
		t.Fatalf("cycle dirty = %v, %t; want second", dirty, ok)
	}
	commitArtifactTestRevision(
		t,
		graph,
		second,
		artifactCallable("second-a"),
		artifactTestDependency(
			t,
			first,
			api.ArtifactFacetCallableSignature,
		),
	)
	if graph.HasPending() {
		t.Fatal("stable cycle did not converge")
	}

	err := graph.Commit(
		artifactTestOwner(first),
		artifactCallable("first-a"),
		nil,
	)
	var convergenceError *ArtifactConvergenceError
	if !errors.As(err, &convergenceError) ||
		convergenceError.Object != artifactTestOwner(first) ||
		len(convergenceError.Facets) != 1 ||
		convergenceError.Facets[0] != api.ArtifactFacetCallableSignature {
		t.Fatalf("oscillation error = %#v", err)
	}
}

func commitArtifactTestRevision(
	t *testing.T,
	graph *Graph,
	owner types.Object,
	contract Contract,
	dependencies ...api.ArtifactDependency,
) {
	t.Helper()
	if err := graph.Commit(
		artifactTestOwner(owner),
		contract,
		dependencies,
	); err != nil {
		t.Fatal(err)
	}
}

func artifactCallable(value string) Contract {
	contract, err := NewContractFacet(
		api.ArtifactFacetCallableSignature,
		[]byte(value),
	)
	if err != nil {
		panic(err)
	}
	return contract
}

func artifactTestContract(
	t *testing.T,
	facets map[api.ArtifactFacet]string,
) Contract {
	t.Helper()
	contract := NewContract()
	var err error
	for facet, value := range facets {
		contract, err = contract.WithFacet(facet, []byte(value))
		if err != nil {
			t.Fatal(err)
		}
	}
	return contract
}

func artifactTestContractBytes(
	t *testing.T,
	facet api.ArtifactFacet,
	value []byte,
) Contract {
	t.Helper()
	contract, err := NewContractFacet(facet, value)
	if err != nil {
		t.Fatal(err)
	}
	return contract
}

func artifactTestDependency(
	t *testing.T,
	provider types.Object,
	facet api.ArtifactFacet,
) api.ArtifactDependency {
	t.Helper()
	dependency, err := api.NewArtifactDependency(
		artifactTestOwner(provider),
		facet,
	)
	if err != nil {
		t.Fatal(err)
	}
	return dependency
}

func artifactTestObject(name string, position token.Pos) types.Object {
	return types.NewFunc(
		position,
		types.NewPackage("example.com/artifacts", "artifacts"),
		name,
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
}

func artifactTestOwner(object types.Object) api.ArtifactOwner {
	return api.MustSourceArtifactOwner(object)
}

func compareArtifactTestObjects(
	left api.ArtifactOwner,
	right api.ArtifactOwner,
) int {
	leftObject, leftOK := left.Source()
	rightObject, rightOK := right.Source()
	if !leftOK || !rightOK {
		panic("artifact graph test owner is not source-backed")
	}
	return compareArtifactTestSourceObjects(leftObject, rightObject)
}

func compareArtifactTestSourceObjects(left types.Object, right types.Object) int {
	leftPackage := ""
	if left.Pkg() != nil {
		leftPackage = left.Pkg().Path()
	}
	rightPackage := ""
	if right.Pkg() != nil {
		rightPackage = right.Pkg().Path()
	}
	switch {
	case leftPackage < rightPackage:
		return -1
	case leftPackage > rightPackage:
		return 1
	case left.Pos() < right.Pos():
		return -1
	case left.Pos() > right.Pos():
		return 1
	case left.Name() < right.Name():
		return -1
	case left.Name() > right.Name():
		return 1
	default:
		return 0
	}
}

func artifactTestNames(objects []api.ArtifactOwner) []string {
	names := make([]string, len(objects))
	for index, owner := range objects {
		names[index] = owner.Name()
	}
	return names
}
