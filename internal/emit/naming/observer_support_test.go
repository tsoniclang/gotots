package naming

import (
	"go/ast"
	"go/types"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// stubEnvironmentObserver satisfies the non-optional environment observer
// contract for focused naming tests that exercise selection without a
// program session.
type stubEnvironmentObserver struct {
	requireUse func(
		types.Object,
		environmentcontract.UseDemand,
		gostdlib.UseSelection,
	) error
}

func (o stubEnvironmentObserver) RequireUse(
	object types.Object,
	demand environmentcontract.UseDemand,
	selection gostdlib.UseSelection,
) error {
	if o.requireUse == nil {
		return nil
	}
	return o.requireUse(object, demand, selection)
}

func (o stubEnvironmentObserver) ObserveImplementation(
	types.Object,
	environmentcontract.UseDemand,
	environmentcontract.ImplementationRoute,
) error {
	return nil
}

// TestForFileRejectsAbsentObserver proves the fail-open route is gone: a
// file name owner cannot exist, and therefore no provider target can be
// returned, without the root environment observer.
func TestForFileRejectsAbsentObserver(t *testing.T) {
	if _, err := NewOwner(nil, nil, NewRegistry()).ForFile(
		nil,
		nil,
		tsgo.NewFactory(),
		"modules/application/source.ts",
		nil,
	); err == nil {
		t.Fatal("file name owner accepted an absent environment observer")
	}
}

func testFileNames(
	t *testing.T,
	owner *Owner,
	sourceFile *ast.File,
	scope *types.Scope,
	factory tsgo.Factory,
	targetPath string,
	observer EnvironmentObserver,
) *File {
	t.Helper()
	if observer == nil {
		observer = stubEnvironmentObserver{}
	}
	names, err := owner.ForFile(
		sourceFile,
		scope,
		factory,
		targetPath,
		observer,
	)
	if err != nil {
		t.Fatal(err)
	}
	return names.(*File)
}
