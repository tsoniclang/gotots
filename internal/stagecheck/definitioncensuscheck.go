package stagecheck

import (
	"github.com/tsoniclang/gotots/internal/language/structure"
)

func compareDefinitionCensus(
	pkg structure.PackageGraph,
	records []structure.DefinitionCensusRecord,
) error {
	actual := newStructuralLedger()
	for _, record := range records {
		actual.add(
			"definition-census",
			record.Package().String()+"|"+record.ID().String(),
		)
	}
	expected := newStructuralLedger()
	for _, definition := range pkg.Definitions() {
		expected.add(
			"definition-census",
			pkg.ID().String()+"|"+definition.ID().String(),
		)
	}
	return compareLedgers(
		"definition-census/"+pkg.ID().String(),
		actual,
		expected,
	)
}
