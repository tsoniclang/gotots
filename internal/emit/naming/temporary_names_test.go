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
		"__gotots_field_0",
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
			"__gotots_field_1": {},
		},
	}

	name, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil || name != "__gotots_field_2" {
		t.Fatalf("temporary = %q, %v; want __gotots_field_2", name, err)
	}
	if imported := file.allocateProviderImportName(name); imported == name {
		t.Fatalf("provider import reused generated binding %q", name)
	}
}

func TestTemporarySnapshotRestoresAllocatedNameSet(t *testing.T) {
	file := &File{
		owner:          newNameOwner(nil, nil),
		temporaries:    make(map[api.TemporaryKind]uint64),
		generatedNames: make(map[string]struct{}),
		importNames:    make(map[string]struct{}),
	}
	start := file.SnapshotTemporaries()
	first, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil {
		t.Fatal(err)
	}
	current := file.SnapshotTemporaries()

	file.RestoreTemporaries(start)
	replayed, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil || replayed != first {
		t.Fatalf("replayed temporary = %q, %v; want %q", replayed, err, first)
	}
	file.RestoreTemporaries(current)
	next, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil || next == first {
		t.Fatalf("next temporary = %q, %v; must differ from %q", next, err, first)
	}
}

func TestTemporaryReplayRetainsNewlyAllocatedNames(t *testing.T) {
	file := &File{
		owner:          newNameOwner(nil, nil),
		temporaries:    make(map[api.TemporaryKind]uint64),
		generatedNames: make(map[string]struct{}),
		importNames:    make(map[string]struct{}),
	}
	start := file.SnapshotTemporaries()
	first, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil {
		t.Fatal(err)
	}
	current := file.SnapshotTemporaries()

	file.RestoreTemporaries(start)
	replayed, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil || replayed != first {
		t.Fatalf("replayed temporary = %q, %v; want %q", replayed, err, first)
	}
	additional, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil {
		t.Fatal(err)
	}
	file.FinishTemporaryReplay(current)

	if imported := file.allocateProviderImportName(additional); imported == additional {
		t.Fatalf("provider import reused replay-added binding %q", additional)
	}
	next, err := file.Temporary(api.TemporaryCompositeField)
	if err != nil || next == first || next == additional {
		t.Fatalf("next temporary = %q, %v; collides with replayed names", next, err)
	}
}
