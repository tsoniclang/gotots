package reflectvalue_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestReflectDescriptorAssignmentCanonicalizesExactProviderOperations(test *testing.T) {
	source, err := os.ReadFile(filepath.Join(repositoryRoot(), "testdata/constructs/value/providerstorage/source.go"))
	if err != nil {
		test.Fatal(err)
	}
	verifyReflectCanonicalInspect(test, string(source), "Descriptors", "providerstorage",
		`console.log(Descriptors());`,
		`package main
import ("fmt"; fixture "example.com/reflectvalue")
func main() { fmt.Println(fixture.Descriptors()) }
`, func(artifacts renderedArtifacts) {
			if !strings.Contains(artifacts.printed, "ReflectValueOperations.$assign") ||
				!strings.Contains(artifacts.printed, "ReflectValueOperations.$copy") {
				test.Fatal("provider assignment or independent descriptor copy was omitted")
			}
		})
}

func TestReflectDescriptorCopyRetainsLiveLocation(test *testing.T) {
	source, err := os.ReadFile(filepath.Join(repositoryRoot(), "testdata/constructs/value/providerstorage/source.go"))
	if err != nil {
		test.Fatal(err)
	}
	verifyReflectCanonicalInspect(test, string(source), "LiveLocations", "providerstorage",
		`console.log(LiveLocations());`,
		`package main
import ("fmt"; fixture "example.com/reflectvalue")
func main() { fmt.Println(fixture.LiveLocations()) }
`, func(artifacts renderedArtifacts) {
			if !strings.Contains(artifacts.printed, ".CanAddr()") {
				test.Fatal("provider addressability contract was omitted")
			}
		})
}

func TestIndirectMutationDoesNotRefineScalarStorage(test *testing.T) {
	source, err := os.ReadFile(filepath.Join(repositoryRoot(), "testdata/constructs/value/providerstorage/source.go"))
	if err != nil {
		test.Fatal(err)
	}
	verifyReflectCanonicalInspect(test, string(source), "MutationConditions", "providerstorage",
		`console.log(MutationConditions());`,
		`package main
import ("fmt"; fixture "example.com/reflectvalue")
func main() { fmt.Println(fixture.MutationConditions()) }
`, nil)
}

func TestProviderAssignmentClosure(test *testing.T) {
	source, err := os.ReadFile(filepath.Join(repositoryRoot(), "testdata/constructs/value/providerstorage/source.go"))
	if err != nil {
		test.Fatal(err)
	}
	for _, function := range []string{"SyncReset", "AtomicReset", "MemStatsFields", "StructFields", "MetricsFields"} {
		test.Run(function, func(test *testing.T) {
			profile := emit.IntegerRepresentationNumber
			if function == "MemStatsFields" {
				profile = emit.IntegerRepresentationFixed64BigInt
			}
			verifyReflectCanonicalInspect(test, string(source), function, "providerstorage",
				"console.log("+function+"());",
				"package main\nimport (\"fmt\"; fixture \"example.com/reflectvalue\")\nfunc main() { fmt.Println(fixture."+function+"()) }\n",
				nil, profile)
		})
	}
}

func TestProviderAggregateCarrierMismatchFailsBeforePublication(test *testing.T) {
	source, err := os.ReadFile(filepath.Join(repositoryRoot(), "testdata/constructs/value/providerstorage/source.go"))
	if err != nil {
		test.Fatal(err)
	}
	_, err = tryCompileReflectFixture(test, test.TempDir(), string(source), []string{"MemStatsFields"})
	var boundary *api.UnsupportedError
	if !errors.As(err, &boundary) || !strings.Contains(boundary.Construct, "provider aggregate [256]uint64 requires an alias-preserving scalar-ABI projection") {
		test.Fatalf("expected exact provider array transport boundary, got %v", err)
	}
}
