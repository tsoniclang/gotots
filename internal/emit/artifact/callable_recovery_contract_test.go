package artifact

import (
	"bytes"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestCallableRecoveryIsOneExactIndependentFacet(t *testing.T) {
	factory := tsgo.NewFactory()
	baseline, err := ProjectCoverageContract(factory, nil)
	if err != nil {
		t.Fatal(err)
	}
	ordinary, err := WithCallableRecovery(baseline, factory, false)
	if err != nil {
		t.Fatal(err)
	}
	recovery, err := WithCallableRecovery(baseline, factory, true)
	if err != nil {
		t.Fatal(err)
	}
	ordinaryValue, ordinaryOK := ordinary.facet(
		api.ArtifactFacetCallableRecovery,
	)
	recoveryValue, recoveryOK := recovery.facet(
		api.ArtifactFacetCallableRecovery,
	)
	if !ordinaryOK || !recoveryOK ||
		bytes.Equal(ordinaryValue, recoveryValue) {
		t.Fatal("callable recovery states did not produce distinct exact facets")
	}
	exportValue, _ := ordinary.facet(api.ArtifactFacetExportSurface)
	recoveryExportValue, _ := recovery.facet(api.ArtifactFacetExportSurface)
	if !bytes.Equal(exportValue, recoveryExportValue) {
		t.Fatal("callable recovery changed the source export surface")
	}
}
