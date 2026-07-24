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
		addRecord(
			&actual.definitionCensus,
			definitionCensusLedgerRecord{
				pkg: record.Package(), definition: record.ID(),
			},
		)
	}
	expected := newStructuralLedger()
	for _, definition := range pkg.Definitions() {
		addRecord(
			&expected.definitionCensus,
			definitionCensusLedgerRecord{
				pkg: pkg.ID(), definition: definition.ID(),
			},
		)
	}
	return compareLedgers(
		"definition-census/"+pkg.ID().String(),
		actual,
		expected,
	)
}
