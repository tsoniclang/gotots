package sourceinvocation

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestRuntimeProjectionIsCanonicalAndImmutable(t *testing.T) {
	scalar, err := api.NewScalarABI(
		api.IntegerRepresentationNumber,
		api.NativeIntegerWidth64,
	)
	if err != nil {
		t.Fatal(err)
	}
	assembled, err := runtimeemission.AssemblePackage(
		tsgo.NewFactory(),
		scalar,
		api.ConcurrencySemanticsDisabled,
		map[api.RuntimeSymbol]struct{}{
			api.RuntimeArrayAllocate:   {},
			api.RuntimeInterfaceNonNil: {},
			api.RuntimeStringIndex:     {},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := FromRuntime(assembled)
	if err != nil {
		t.Fatal(err)
	}
	files := contract.Files()
	if len(files) < 3 {
		t.Fatalf("source invocation files = %#v", files)
	}
	for index := 1; index < len(files); index++ {
		if files[index-1].SourcePath() >= files[index].SourcePath() {
			t.Fatalf("source invocation files are not canonical: %#v", files)
		}
	}
	invocations := contract.Invocations()
	if len(invocations) == 0 {
		t.Fatal("source invocation contract has no invocations")
	}
	for index := 1; index < len(invocations); index++ {
		if compareInvocation(invocations[index-1], invocations[index]) >= 0 {
			t.Fatalf("source invocations are not canonical: %#v", invocations)
		}
	}
	files[0].sourcePath = "mutated.ts"
	if contract.Files()[0].SourcePath() == "mutated.ts" {
		t.Fatal("source invocation contract exposes file storage")
	}
	for index := range invocations {
		origins := invocations[index].ResultOriginParameters()
		if len(origins) == 0 {
			continue
		}
		origins[0] = 9
		if contract.Invocations()[index].ResultOriginParameters()[0] == 9 {
			t.Fatal("source invocation contract exposes parameter-index storage")
		}
		return
	}
	t.Fatal("source invocation fixture has no result-origin evidence")
}
