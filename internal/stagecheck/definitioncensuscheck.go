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
	if err := pkg.VisitDefinitions(func(
		definition structure.ImplementationDefinition,
	) error {
		addRecord(
			&expected.definitionCensus,
			definitionCensusLedgerRecord{
				pkg: pkg.ID(), definition: definition.ID(),
			},
		)
		return nil
	}); err != nil {
		return err
	}
	return compareLedgers(
		"definition-census/"+pkg.ID().String(),
		actual,
		expected,
	)
}
