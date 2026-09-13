package slice_test

import (
	"strings"
	"testing"

	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestSliceArrayPointerDoesNotReadOrWriteItsBase(test *testing.T) {
	factory := tsgo.NewFactory()
	declaration := runtimeslice.BuildArrayPointer(factory, "arrayPointer", "RuntimeSlice", "Pointer", "viewPointer", "GoArray", "goArrayFromRegion")
	client, err := tsgo.StartClient(repositoryRoot(), test.TempDir())
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() { _ = client.Close() })
	printed, err := client.PrintNode(declaration, tsgo.PrintOptions{})
	if err != nil {
		test.Fatal(err)
	}
	for _, required := range []string{"viewPointer<T, GoArray<T, N>>", "goRegionAddress<T>(location, 0)", "(): GoArray<T, N> => view", "target: GoArray<T, N>", "): void =>"} {
		if !strings.Contains(printed, required) {
			test.Fatalf("array view lacks %q", required)
		}
	}
	for _, forbidden := range []string{"projectPointer", "goRegionRead", "loadPointer", "storePointer", "_source"} {
		if strings.Contains(printed, forbidden) {
			test.Fatalf("array view accesses a nonexistent base through %q", forbidden)
		}
	}
}
