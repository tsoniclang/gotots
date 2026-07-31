package artifact

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestObservableContractRejectsGenericEmptyButAcceptsTypedCoverage(t *testing.T) {
	factory := tsgo.NewFactory()
	if _, err := ProjectContract(factory, nil); err == nil {
		t.Fatal("generic empty materialized contract was accepted")
	}
	contract, err := ProjectCoverageContract(factory, nil)
	if err != nil {
		t.Fatal(err)
	}
	exports, ok := contract.ExportedBindings()
	if !contract.initialized ||
		!contract.hasFacet(api.ArtifactFacetExportSurface) ||
		!ok ||
		len(exports) != 0 {
		t.Fatalf("coverage contract = %#v, want explicit empty export surface", contract)
	}
	statement := artifactTestFunction(factory, "value", nil)
	if _, err := ProjectCoverageContract(
		factory,
		[]tsgo.Statement{statement},
	); err == nil {
		t.Fatal("coverage-only contract accepted a target declaration")
	}
	if _, err := ProjectFacet(
		api.ArtifactFacetExportSurface,
		factory.NamedExports(nil),
	); err == nil {
		t.Fatal("standalone export surface bypassed declaration projection")
	}
}
