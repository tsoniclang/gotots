package naming

import (
	"go/ast"
	"go/token"
	"go/types"
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

func temporaryTestOwner(name string) api.ArtifactOwner {
	return api.MustSourceArtifactOwner(types.NewVar(
		token.NoPos,
		types.NewPackage("example/"+name, "example"),
		name,
		types.Typ[types.Int],
	))
}
