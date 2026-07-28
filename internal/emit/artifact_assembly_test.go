package emit

import (
	"bytes"
	"errors"
	"go/token"
	"go/types"
	"sort"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestCommittedArtifactPlacementDropsSupersededImports(t *testing.T) {
	factory := tsgo.NewFactory()
	base := artifactTestImport(t, factory, "./base.js", "Base")
	old := artifactTestImport(t, factory, "./old.js", "Old")
	next := artifactTestImport(t, factory, "./next.js", "Next")
	basePlacement := targetplacement.New()
	if err := basePlacement.Apply([]api.RootRequest{base}); err != nil {
		t.Fatal(err)
	}
	oldPlacement := targetplacement.New()
	if err := oldPlacement.Apply([]api.RootRequest{old}); err != nil {
		t.Fatal(err)
	}
	builder := &targetFileBuilder{
		placement: basePlacement,
		declarations: []targetDeclaration{{
			owner:     sourceArtifactOwner(artifactTestObject("Consumer", 10)),
			placement: oldPlacement,
		}},
	}
	committed, err := committedTargetFilePlacement(builder)
	if err != nil {
		t.Fatal(err)
	}
	if actual := artifactTestModules(committed); !equalStrings(
		actual,
		[]string{"./base.js", "./old.js"},
	) {
		t.Fatalf("initial modules = %v", actual)
	}

	nextPlacement := targetplacement.New()
	if err := nextPlacement.Apply([]api.RootRequest{next}); err != nil {
		t.Fatal(err)
	}
	builder.declarations[0].placement = nextPlacement
	committed, err = committedTargetFilePlacement(builder)
	if err != nil {
		t.Fatal(err)
	}
	if actual := artifactTestModules(committed); !equalStrings(
		actual,
		[]string{"./base.js", "./next.js"},
	) {
		t.Fatalf("replacement modules = %v", actual)
	}
}

func TestArtifactReconstructionReplaysTemporaryNamespace(t *testing.T) {
	program := loadDeclarationAssemblyFixture(t)
	trigger := program.Roots()[0].Types().Scope().Lookup("Trigger")
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := session.require(trigger); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	declaration := declarationForObject(t, session, trigger)
	before, err := tsgo.EncodeNode(
		session.factory.SyntaxList(statementsAsNodes(declaration.statements)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.reconstructArtifact(trigger); err != nil {
		t.Fatal(err)
	}
	after, err := tsgo.EncodeNode(
		session.factory.SyntaxList(statementsAsNodes(declaration.statements)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("unchanged artifact reconstruction changed temporary identities")
	}
	if session.artifacts.HasPending() {
		t.Fatal("unchanged reconstruction dirtied another artifact")
	}
}

func TestTargetFilesRejectPendingArtifactReconstruction(t *testing.T) {
	program := loadDeclarationAssemblyFixture(t)
	caller := program.Roots()[0].Types().Scope().Lookup("Caller")
	trigger := program.Roots()[0].Types().Scope().Lookup("Trigger")
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := session.require(caller); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	if err := session.artifacts.Commit(
		sourceArtifactOwner(trigger),
		artifactstate.Contract{
			api.ArtifactFacetCallableSignature: []byte("changed"),
		},
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if dirty, ok := session.artifacts.NextDirty(); !ok ||
		dirty != sourceArtifactOwner(caller) {
		t.Fatalf("dirty = %v, %t; want Caller", dirty, ok)
	} else if err := session.artifacts.Commit(
		sourceArtifactOwner(trigger),
		artifactstate.Contract{
			api.ArtifactFacetCallableSignature: []byte("changed-again"),
		},
		nil,
	); err != nil {
		t.Fatal(err)
	}
	_, err = session.targetFiles()
	var scheduleError *ScheduleError
	if !errors.As(err, &scheduleError) {
		t.Fatalf("seal error = %#v, want ScheduleError", err)
	}
	if session.sealed {
		t.Fatal("failed dirty-artifact seal closed the session")
	}
}

func TestNonArtifactDependencyCannotBeSilentlyDropped(t *testing.T) {
	provider := artifactTestObject("Provider", 10)
	request, err := api.NewArtifactDependencyRequest(
		provider,
		api.ArtifactFacetCallableSignature,
	)
	if err != nil {
		t.Fatal(err)
	}
	session := &programSession{}
	err = session.applyRootRequests(
		targetplacement.New(),
		[]api.RootRequest{request},
	)
	var scheduleError *ScheduleError
	if !errors.As(err, &scheduleError) {
		t.Fatalf("dependency error = %#v, want ScheduleError", err)
	}
}

func artifactTestImport(
	t *testing.T,
	factory tsgo.Factory,
	module string,
	name string,
) api.RootRequest {
	t.Helper()
	request, err := api.NewImportRequest(
		factory,
		api.ImportPhaseValue,
		module,
		name,
		name,
	)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func artifactTestModules(placement *targetplacement.Owner) []string {
	modules := make([]string, 0)
	for _, request := range placement.Requests() {
		modules = append(modules, request.ModulePath())
	}
	sort.Strings(modules)
	return modules
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func statementsAsNodes(statements []tsgo.Statement) []tsgo.Node {
	nodes := make([]tsgo.Node, len(statements))
	for index, statement := range statements {
		nodes[index] = statement
	}
	return nodes
}

func TestSchedulerDeduplicatesPendingAndCycleReferences(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/schedule", "schedule")
	first := types.NewFunc(
		token.Pos(1),
		sourcePackage,
		"First",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	second := types.NewFunc(
		token.Pos(2),
		sourcePackage,
		"Second",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	scheduler := newScheduler()
	scheduler.enqueue(first)
	scheduler.enqueue(first)
	if object, ok := scheduler.next(); !ok || object != first {
		t.Fatalf("first scheduled object = %v, %v", object, ok)
	}
	scheduler.enqueue(second)
	scheduler.enqueue(first)
	if object, ok := scheduler.next(); !ok || object != second {
		t.Fatalf("second scheduled object = %v, %v", object, ok)
	}
	if object, ok := scheduler.next(); ok || object != nil {
		t.Fatalf("duplicate cycle target was re-enqueued: %v", object)
	}
}

func TestDeclarationRequirementSchedulerDeduplicatesAndUsesClosedOrder(
	t *testing.T,
) {
	sourcePackage := types.NewPackage("example.com/schedule", "schedule")
	first := types.NewTypeName(token.Pos(1), sourcePackage, "First", nil)
	second := types.NewTypeName(token.Pos(2), sourcePackage, "Second", nil)
	firstCopy, err := api.NewNamedStructOperationRequirement(
		first,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	firstEqual, err := api.NewNamedStructOperationRequirement(
		first,
		api.NamedStructOperationEqual,
	)
	if err != nil {
		t.Fatal(err)
	}
	secondZero, err := api.NewNamedStructOperationRequirement(
		second,
		api.NamedStructOperationZero,
	)
	if err != nil {
		t.Fatal(err)
	}

	scheduler := newDeclarationRequirementScheduler()
	scheduler.enqueue(secondZero)
	scheduler.enqueue(firstEqual)
	scheduler.enqueue(firstCopy)
	scheduler.enqueue(firstCopy)

	firstBatch, ok := scheduler.nextBatch()
	if !ok ||
		len(firstBatch) != 2 ||
		firstBatch[0] != firstCopy ||
		firstBatch[1] != firstEqual {
		t.Fatalf("first requirement batch = %#v, %t", firstBatch, ok)
	}
	secondBatch, ok := scheduler.nextBatch()
	if !ok || len(secondBatch) != 1 || secondBatch[0] != secondZero {
		t.Fatalf("second requirement batch = %#v, %t", secondBatch, ok)
	}
	if actual, ok := scheduler.nextBatch(); ok || actual != nil {
		t.Fatalf("unexpected trailing requirement batch = %#v, %t", actual, ok)
	}
}

func artifactTestObject(name string, position token.Pos) types.Object {
	return types.NewFunc(
		position,
		types.NewPackage("example.com/artifacts", "artifacts"),
		name,
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
}
