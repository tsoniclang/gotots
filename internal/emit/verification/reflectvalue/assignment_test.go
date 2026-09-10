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
