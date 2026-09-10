package reflectvalue_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
