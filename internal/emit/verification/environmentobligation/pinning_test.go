package environmentobligation_test

import (
	"context"
	"errors"
	"go/types"
	"path/filepath"
	"testing"

	environmentidentity "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestSelectedJavaScriptProviderRejectsUnimplementedPinSets(test *testing.T) {
	project := test.TempDir()
	writeProgramFile(test, filepath.Join(project, "go.mod"), "module example.com/pinsets\n\ngo 1.26.4\n")
	writeProgramFile(test, filepath.Join(project, "source.go"), `package pinsets
import "runtime"

func Direct(pinner *runtime.Pinner, value *uint32) { pinner.Pin(value) }
func Bound(pinner *runtime.Pinner) func(any) { return pinner.Pin }
func Expression() func(*runtime.Pinner, any) { return (*runtime.Pinner).Pin }
func Deferred(pinner *runtime.Pinner) { defer pinner.Unpin() }
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project, Pattern: ".", BuildProfile: linkedProviderBuildProfile(test),
	})
	if err != nil {
		test.Fatal(err)
	}
	provider := linkedProviderCertificate(test)
	runtimePackage := program.Roots()[0].Types().Imports()[0]
	pinner, ok := runtimePackage.Scope().Lookup("Pinner").(*types.TypeName)
	if !ok {
		test.Fatal("selected runtime has no Pinner type")
	}
	pinSetContract, err := environmentidentity.Describe(pinner)
	if err != nil {
		test.Fatal(err)
	}
	methods := types.NewMethodSet(types.NewPointer(pinner.Type()))
	for _, selected := range []struct{ root, method string }{
		{"Direct", "Pin"}, {"Bound", "Pin"}, {"Expression", "Pin"}, {"Deferred", "Unpin"},
	} {
		test.Run(selected.root, func(test *testing.T) {
			method := methods.Lookup(runtimePackage, selected.method)
			if method == nil {
				test.Fatal("selected pin operation does not exist")
			}
			contract, err := environmentidentity.Describe(method.Obj())
			if err != nil {
				test.Fatal(err)
			}
			root, err := emit.NewRoot(program.Roots()[0].Types().Scope().Lookup(selected.root))
			if err != nil {
				test.Fatal(err)
			}
			options := emit.DefaultOptions()
			options.IntegerRepresentation = emit.IntegerRepresentationNumber
			options.EvaluationOrder = emit.EvaluationOrderDirect
			canonical, err := emit.CompileWithOptions(program, []emit.Root{root}, options)
			if err != nil {
				test.Fatalf("source pin contract was not preserved: %v", err)
			}
			preserved := false
			for _, obligation := range canonical.EnvironmentObligations() {
				if obligation.Identity() == contract.Identity() {
					preserved = obligation.SourceSignature() == contract.Signature() &&
						obligation.Route() == environmentidentity.RouteBoundary
				}
			}
			if !preserved {
				test.Fatal("pin contract was erased or reassigned before provider selection")
			}
			options.StandardLibrary = provider
			_, err = emit.CompileWithOptions(program, []emit.Root{root}, options)
			var missing *api.NameError
			if !errors.As(err, &missing) {
				test.Fatalf("pin set escaped selected-provider binding validation: %v", err)
			}
			if missing.Name != pinSetContract.Identity() || missing.Reason != "selected standard-library declaration has no provider binding" {
				test.Fatalf("missing exact pin-set binding diagnostic: %v", missing)
			}
		})
	}
}
