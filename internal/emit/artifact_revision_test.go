package emit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/load"
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
	if err := session.RequireUse(trigger, rootUseDemand(trigger), gostdlib.NoUseSelection()); err != nil {
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
	if err := session.RequireUse(caller, rootUseDemand(caller), gostdlib.NoUseSelection()); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	if err := session.artifacts.Commit(
		sourceArtifactOwner(trigger),
		artifactTestFacetContract(
			t,
			api.ArtifactFacetCallableSignature,
			"changed",
		),
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if dirty, ok := session.artifacts.NextDirty(); !ok ||
		dirty != sourceArtifactOwner(caller) {
		t.Fatalf("dirty = %v, %t; want Caller", dirty, ok)
	} else if err := session.artifacts.Commit(
		sourceArtifactOwner(trigger),
		artifactTestFacetContract(
			t,
			api.ArtifactFacetCallableSignature,
			"changed-again",
		),
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

func artifactTestFacetContract(
	t *testing.T,
	facet api.ArtifactFacet,
	value string,
) artifactstate.Contract {
	t.Helper()
	contract, err := artifactstate.NewContractFacet(facet, []byte(value))
	if err != nil {
		t.Fatal(err)
	}
	return contract
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

	scheduler := newDeclarationRequirementScheduler(
		emitordering.CompareArtifactOwners,
	)
	scheduler.enqueue(secondZero)
	scheduler.enqueue(firstEqual)
	scheduler.enqueue(firstCopy)
	scheduler.enqueue(firstCopy)

	firstOwner, firstBatch, removed, ok := scheduler.nextBatch()
	if !ok ||
		removed ||
		firstOwner != firstCopy.Owner() ||
		len(firstBatch) != 2 ||
		firstBatch[0] != firstCopy ||
		firstBatch[1] != firstEqual {
		t.Fatalf("first requirement batch = %#v, %t", firstBatch, ok)
	}
	secondOwner, secondBatch, removed, ok := scheduler.nextBatch()
	if !ok ||
		removed ||
		secondOwner != secondZero.Owner() ||
		len(secondBatch) != 1 ||
		secondBatch[0] != secondZero {
		t.Fatalf("second requirement batch = %#v, %t", secondBatch, ok)
	}
	if owner, actual, removed, ok := scheduler.nextBatch(); ok ||
		removed ||
		owner.Valid() ||
		actual != nil {
		t.Fatalf("unexpected trailing requirement batch = %#v, %t", actual, ok)
	}
}

func TestDeclarationRequirementSchedulerReplacesConsumerRevision(
	t *testing.T,
) {
	sourcePackage := types.NewPackage("example.com/revisions", "revisions")
	provider := types.NewTypeName(token.Pos(1), sourcePackage, "Record", nil)
	firstConsumer := api.MustSourceArtifactOwner(types.NewFunc(
		token.Pos(2),
		sourcePackage,
		"First",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	))
	secondConsumer := api.MustSourceArtifactOwner(types.NewFunc(
		token.Pos(3),
		sourcePackage,
		"Second",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	))
	requirement, err := api.NewNamedStructOperationRequirement(
		provider,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	scheduler := newDeclarationRequirementScheduler(
		emitordering.CompareArtifactOwners,
	)
	scheduler.replace(firstConsumer, []api.DeclarationRequirement{requirement})
	scheduler.replace(secondConsumer, []api.DeclarationRequirement{requirement})
	owner, selected, removed, ok := scheduler.nextBatch()
	if !ok ||
		removed ||
		owner != requirement.Owner() ||
		len(selected) != 1 ||
		selected[0] != requirement {
		t.Fatalf("initial replacement batch = %v %#v %t", owner, selected, ok)
	}
	scheduler.replace(firstConsumer, nil)
	if scheduler.hasPending() {
		t.Fatal("shared requirement was removed with one remaining consumer")
	}
	scheduler.replace(secondConsumer, nil)
	if !scheduler.finalizeRemovals() {
		t.Fatal("final requirement removal was not scheduled at quiescence")
	}
	owner, selected, removed, ok = scheduler.nextBatch()
	if !ok || !removed ||
		owner != requirement.Owner() ||
		len(selected) != 0 {
		t.Fatalf("removal batch = %v %#v %t", owner, selected, ok)
	}
	if scheduler.wasApplied(requirement) {
		t.Fatal("removed requirement survived in the applied snapshot")
	}
}

func TestDeclarationRequirementSchedulerReplacesSelfDemand(
	t *testing.T,
) {
	sourcePackage := types.NewPackage("example.com/self-demand", "selfdemand")
	record := types.NewTypeName(token.Pos(1), sourcePackage, "Record", nil)
	requirement, err := api.NewNamedStructOperationRequirement(
		record,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	scheduler := newDeclarationRequirementScheduler(
		emitordering.CompareArtifactOwners,
	)
	scheduler.replace(
		requirement.Owner(),
		[]api.DeclarationRequirement{requirement},
	)
	if _, _, removed, ok := scheduler.nextBatch(); !ok || removed {
		t.Fatal("self demand was not scheduled")
	}
	scheduler.replace(requirement.Owner(), nil)
	if !scheduler.finalizeRemovals() {
		t.Fatal("stale self demand was retained after exact replacement")
	}
	owner, requirements, removed, ok := scheduler.nextBatch()
	if !ok || !removed || owner != requirement.Owner() ||
		len(requirements) != 0 || scheduler.wasApplied(requirement) {
		t.Fatal("stale self demand survived its replacement revision")
	}
}

func TestDeclarationRequirementSchedulerLookupVisitsOnlySelectedOwner(
	t *testing.T,
) {
	const ownerCount = 4096
	sourcePackage := types.NewPackage("example.com/schedule-scale", "schedule")
	scheduler := newDeclarationRequirementScheduler(
		emitordering.CompareArtifactOwners,
	)
	owners := make([]api.ArtifactOwner, 0, ownerCount)
	for index := range ownerCount {
		typeName := types.NewTypeName(
			token.Pos(index+1),
			sourcePackage,
			fmt.Sprintf("Record%d", index),
			nil,
		)
		requirement, err := api.NewNamedStructOperationRequirement(
			typeName,
			api.NamedStructOperationCopy,
		)
		if err != nil {
			t.Fatal(err)
		}
		owners = append(owners, requirement.Owner())
		scheduler.enqueue(requirement)
	}
	for {
		if _, _, _, ok := scheduler.nextBatch(); !ok {
			break
		}
	}
	requirements, visits := scheduler.applied.forOwner(owners[ownerCount/2])
	if len(requirements) != 1 || visits != 1 {
		t.Fatalf(
			"selected-owner lookup returned %d requirements after %d visits",
			len(requirements),
			visits,
		)
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

func sourceArtifactOwner(object types.Object) api.ArtifactOwner {
	return api.MustSourceArtifactOwner(object)
}

func TestAnonymousStructDemandsUseExistingArtifactFixedPoint(t *testing.T) {
	program := loadAnonymousStructAssemblyFixture(t)
	scope := program.Roots()[0].Types().Scope()
	definition := scope.Lookup("Definition").(*types.Func)
	copyValue := scope.Lookup("CopyValue").(*types.Func)
	equal := scope.Lookup("Equal").(*types.Func)
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	if err := session.RequireUse(definition, rootUseDemand(definition), gostdlib.NoUseSelection()); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	artifact := onlyAnonymousStructArtifact(t, session)
	owner := api.MustGeneratedArtifactOwner(artifact)
	initialStaticRevision := session.artifacts.FacetRevision(
		owner,
		api.ArtifactFacetStaticSurface,
	)
	if _, fabricated := owner.Source(); fabricated {
		t.Fatal("anonymous artifact fabricated a go/types source object")
	}

	if err := session.RequireUse(copyValue, rootUseDemand(copyValue), gostdlib.NoUseSelection()); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	copyStaticRevision := session.artifacts.FacetRevision(
		owner,
		api.ArtifactFacetStaticSurface,
	)
	if copyStaticRevision != initialStaticRevision+1 {
		t.Fatalf(
			"copy static revision = %d, want %d",
			copyStaticRevision,
			initialStaticRevision+1,
		)
	}
	if declarationForObject(t, session, copyValue).reconstructions != 1 {
		t.Fatal("copy consumer did not reconstruct through the existing artifact graph")
	}

	if err := session.RequireUse(equal, rootUseDemand(equal), gostdlib.NoUseSelection()); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	if revision := session.artifacts.FacetRevision(
		owner,
		api.ArtifactFacetStaticSurface,
	); revision != copyStaticRevision+1 {
		t.Fatalf("equal static revision = %d, want %d", revision, copyStaticRevision+1)
	}
	builder := session.builders[artifact.OutputPath()]
	index, ok := builder.indexByOwner[owner]
	if !ok || builder.declarations[index].reconstructions != 2 {
		t.Fatalf("anonymous artifact reconstruction = %#v, %t", builder, ok)
	}
	if session.artifacts.HasPending() ||
		session.requirements.hasPending() ||
		session.scheduler.hasPending() {
		t.Fatal("anonymous-struct requirements did not converge")
	}
}

func loadAnonymousStructAssemblyFixture(t *testing.T) *load.Program {
	t.Helper()
	directory := t.TempDir()
	writeAnonymousAssemblyFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/anonymousassembly\n\ngo 1.26.4\n",
	)
	writeAnonymousAssemblyFile(t, filepath.Join(directory, "source.go"), `package anonymousassembly

func Definition(value struct{ Field int32 }) int32 {
	return value.Field
}

func CopyValue(value struct{ Field int32 }) struct{ Field int32 } {
	copy := value
	return copy
}

func Equal(left, right struct{ Field int32 }) bool {
	return left == right
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func writeAnonymousAssemblyFile(
	t *testing.T,
	path string,
	content string,
) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func onlyAnonymousStructArtifact(
	t *testing.T,
	session *programSession,
) *api.GeneratedArtifact {
	t.Helper()
	artifacts := session.registry.GeneratedArtifacts(
		api.GeneratedArtifactAnonymousStruct,
	)
	if len(artifacts) != 1 {
		t.Fatalf(
			"anonymous artifacts = %d, want one",
			len(artifacts),
		)
	}
	return artifacts[0]
}
