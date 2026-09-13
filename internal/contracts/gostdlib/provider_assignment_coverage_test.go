package gostdlib_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestShippedCopyFacetsHaveStableAssignment(test *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "gostdlib", "contract", "manifest.json"))
	if err != nil {
		test.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		test.Fatal(err)
	}
	checked := 0
	for _, module := range manifest.FacetModules() {
		for _, facet := range module.Facets() {
			if facet.Kind() != gostdlib.FacetNamedStructOperations ||
				!slices.Contains(facet.Capabilities(), gostdlib.FacetCapabilityCopy) {
				continue
			}
			checked++
			if !slices.Contains(facet.Capabilities(), gostdlib.FacetCapabilityAssign) {
				test.Errorf("%s can copy but cannot preserve an assignment destination", facet.SourceIdentity())
			}
		}
	}
	if checked == 0 {
		test.Fatal("no shipped value-copy contracts were checked")
	}
}
