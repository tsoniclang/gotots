package naming

import (
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"reflect"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestTemporaryNameAvoidsSourceImportAndGeneratedBindings(t *testing.T) {
	packageScope := types.NewScope(nil, token.NoPos, token.NoPos, "package")
	functionScope := types.NewScope(packageScope, token.NoPos, token.NoPos, "function")
	reserved := types.NewVar(
		token.NoPos,
		nil,
		"fieldValue",
		types.Typ[types.Int],
	)
	functionScope.Insert(reserved)
	owner := newNameOwner(packageScope, &types.Info{
		Defs: map[*ast.Ident]types.Object{{Name: reserved.Name()}: reserved},
	})
	file := &File{
		owner:          owner,
		temporaries:    make(map[api.TemporaryKind]uint64),
		generatedNames: make(map[string]struct{}),
		importNames: map[string]struct{}{
			"fieldValue2": {},
		},
	}

	name, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil || name != "fieldValue3" {
		t.Fatalf("temporary = %q, %v; want fieldValue3", name, err)
	}
	if imported := file.allocateProviderImportName(name); imported == name {
		t.Fatalf("provider import reused generated binding %q", name)
	}
}

func TestTemporaryReplayReservesOtherArtifactBindings(t *testing.T) {
	file := &File{
		owner:           newNameOwner(nil, nil),
		temporaries:     make(map[api.TemporaryKind]uint64),
		generatedNames:  make(map[string]struct{}),
		temporaryOwners: make(map[string]api.ArtifactOwner),
		importNames:     make(map[string]struct{}),
	}
	firstOwner := temporaryTestOwner("First")
	secondOwner := temporaryTestOwner("Second")
	file.artifactOwner = firstOwner
	start := file.SnapshotTemporaries()
	first, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil {
		t.Fatal(err)
	}
	file.artifactOwner = secondOwner
	second, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil {
		t.Fatal(err)
	}
	finish, err := file.BeginTemporaryReplay(firstOwner, start)
	if err != nil {
		t.Fatal(err)
	}
	file.artifactOwner = firstOwner
	replayed, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil || replayed != first {
		t.Fatalf("replayed temporary = %q, %v; want %q", replayed, err, first)
	}
	additional, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil {
		t.Fatal(err)
	}
	if additional == first || additional == second {
		t.Fatalf(
			"replay allocation %q collides with %q / %q",
			additional,
			first,
			second,
		)
	}
	finish(true)
	if file.temporaryOwners[first] != firstOwner ||
		file.temporaryOwners[second] != secondOwner ||
		file.temporaryOwners[additional] != firstOwner {
		t.Fatal("temporary ownership was not preserved across replay")
	}
}

func TestTemporaryReplayRemovesRetiredArtifactBindings(t *testing.T) {
	file := &File{
		owner:           newNameOwner(nil, nil),
		temporaries:     make(map[api.TemporaryKind]uint64),
		generatedNames:  make(map[string]struct{}),
		temporaryOwners: make(map[string]api.ArtifactOwner),
		importNames:     make(map[string]struct{}),
	}
	owner := temporaryTestOwner("Owner")
	file.artifactOwner = owner
	start := file.SnapshotTemporaries()
	retired, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil {
		t.Fatal(err)
	}
	finish, err := file.BeginTemporaryReplay(owner, start)
	if err != nil {
		t.Fatal(err)
	}
	file.artifactOwner = owner
	replacement, err := file.Temporary(api.TemporaryAssignmentValue)
	if err != nil {
		t.Fatal(err)
	}
	finish(true)
	if _, retained := file.generatedNames[retired]; retained {
		t.Fatalf("retired temporary %q remains reserved", retired)
	}
	if _, retained := file.generatedNames[replacement]; !retained {
		t.Fatalf("replacement temporary %q was not retained", replacement)
	}
}

func TestTemporaryReplayRollsBackFailedReconstruction(t *testing.T) {
	file := &File{
		owner:           newNameOwner(nil, nil),
		temporaries:     make(map[api.TemporaryKind]uint64),
		generatedNames:  make(map[string]struct{}),
		temporaryOwners: make(map[string]api.ArtifactOwner),
		importNames:     make(map[string]struct{}),
	}
	owner := temporaryTestOwner("Owner")
	file.artifactOwner = owner
	start := file.SnapshotTemporaries()
	original, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil {
		t.Fatal(err)
	}
	finish, err := file.BeginTemporaryReplay(owner, start)
	if err != nil {
		t.Fatal(err)
	}
	file.artifactOwner = owner
	partial, err := file.Temporary(api.TemporaryAssignmentValue)
	if err != nil {
		t.Fatal(err)
	}
	finish(false)
	if file.temporaryOwners[original] != owner {
		t.Fatalf("original temporary %q was not restored", original)
	}
	if _, retained := file.generatedNames[partial]; retained {
		t.Fatalf("failed replay temporary %q remains reserved", partial)
	}
}

func TestTemporarySnapshotRetainsOnlyIndependentCounters(t *testing.T) {
	onlyCounters := func(shape reflect.Type) bool {
		return shape.NumField() == 1 && shape.Field(0).Type == reflect.TypeFor[map[api.TemporaryKind]uint64]()
	}
	if !onlyCounters(reflect.TypeFor[TemporarySnapshot]()) {
		t.Fatal("artifact checkpoint retains file-wide state instead of only temporary counters")
	}
	control := reflect.TypeFor[struct {
		counters map[api.TemporaryKind]uint64
		names    map[string]struct{}
	}]()
	if onlyCounters(control) {
		t.Fatal("retention control failed to detect a copied file-wide name registry")
	}
	file := &File{temporaries: map[api.TemporaryKind]uint64{api.TemporaryCompositeField: 7}}
	snapshot := file.SnapshotTemporaries()
	file.temporaries[api.TemporaryCompositeField] = 9
	file.temporaries[api.TemporaryAssignmentValue] = 3
	if len(snapshot.counters) != 1 || snapshot.counters[api.TemporaryCompositeField] != 7 {
		t.Fatal("later allocations mutate the retained counter checkpoint")
	}
}

func TestTemporaryReplayEmptyCheckpointPreservesRollbackState(t *testing.T) {
	owner := temporaryTestOwner("Owner")
	other := temporaryTestOwner("Other")
	file := &File{
		owner:           newNameOwner(nil, nil),
		temporaries:     map[api.TemporaryKind]uint64{api.TemporaryCompositeField: 2},
		generatedNames:  map[string]struct{}{"fieldValue": {}, "fieldValue2": {}},
		temporaryOwners: map[string]api.ArtifactOwner{"fieldValue": owner, "fieldValue2": other},
		importNames:     map[string]struct{}{"fieldValue3": {}},
	}
	originalCounters := maps.Clone(file.temporaries)
	originalNames := maps.Clone(file.generatedNames)
	originalOwners := maps.Clone(file.temporaryOwners)
	finish, err := file.BeginTemporaryReplay(owner, TemporarySnapshot{})
	if err != nil {
		t.Fatal(err)
	}
	file.artifactOwner = owner
	first, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil || first != "fieldValue" {
		t.Fatalf("empty checkpoint replay = %q, %v", first, err)
	}
	second, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil || second != "fieldValue4" {
		t.Fatalf("replay did not reserve other-owner and import bindings: %q, %v", second, err)
	}
	finish(false)
	if !maps.Equal(file.temporaries, originalCounters) || !maps.Equal(file.generatedNames, originalNames) ||
		!maps.Equal(file.temporaryOwners, originalOwners) {
		t.Fatal("rollback changed counters or another artifact's reservations")
	}
}

func temporaryTestOwner(name string) api.ArtifactOwner {
	return api.MustSourceArtifactOwner(types.NewVar(
		token.NoPos,
		types.NewPackage("example/"+name, "example"),
		name,
		types.Typ[types.Int],
	))
}
