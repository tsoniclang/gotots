package semantic

import "testing"

func assertDefinitionRecordsEqual(
	t *testing.T,
	want Package,
	got Package,
) {
	t.Helper()
	var records []DefinitionSemantics
	if err := got.VisitDefinitions(
		func(record DefinitionSemantics) error {
			records = append(records, record)
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	index := 0
	if err := want.VisitDefinitions(
		func(record DefinitionSemantics) error {
			if index >= len(records) ||
				!equalDefinitionRecords(record, records[index]) ||
				record.Authority() != records[index].Authority() {
				t.Fatalf(
					"semantic definition %d differs", index,
				)
			}
			index++
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
}

func assertResolutionRecordsEqual(
	t *testing.T,
	want Package,
	got Package,
) {
	t.Helper()
	var records []OccurrenceResolution
	if err := got.VisitResolutions(
		func(record OccurrenceResolution) error {
			records = append(records, record)
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	index := 0
	if err := want.VisitResolutions(
		func(record OccurrenceResolution) error {
			if index >= len(records) ||
				!equalResolutionRecords(record, records[index]) {
				t.Fatalf(
					"semantic resolution %d differs", index,
				)
			}
			index++
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
}

func assertDeclarationRecordsEqual(
	t *testing.T,
	want Package,
	got Package,
) {
	t.Helper()
	var records []Declaration
	if err := got.VisitDeclarations(
		func(record Declaration) error {
			records = append(records, record)
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	index := 0
	if err := want.VisitDeclarations(
		func(record Declaration) error {
			if index >= len(records) ||
				!equalDeclarationRecords(record, records[index]) ||
				record.Authority() != records[index].Authority() {
				t.Fatalf(
					"semantic declaration %d differs", index,
				)
			}
			index++
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
}

func assertBindingRecordsEqual(
	t *testing.T,
	want Package,
	got Package,
) {
	t.Helper()
	var records []Binding
	if err := got.VisitBindings(
		func(record Binding) error {
			records = append(records, record)
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	index := 0
	if err := want.VisitBindings(
		func(record Binding) error {
			if index >= len(records) ||
				!equalBindingRecords(record, records[index]) ||
				record.Authority() != records[index].Authority() {
				t.Fatalf("semantic binding %d differs", index)
			}
			index++
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
}

func assertTypeRecordsEqual(
	t *testing.T,
	want Package,
	got Package,
) {
	t.Helper()
	var records []Type
	if err := got.VisitTypes(func(record Type) error {
		records = append(records, record)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	index := 0
	if err := want.VisitTypes(func(record Type) error {
		if index >= len(records) ||
			!equalTypeRecords(record, records[index]) {
			t.Fatalf("semantic type %d differs", index)
		}
		wantWitness, wantPresent := want.TypeWitness(record.ID())
		gotWitness, gotPresent := got.TypeWitness(record.ID())
		if !wantPresent || !gotPresent ||
			wantWitness != gotWitness {
			t.Fatalf(
				"semantic type witness %s differs",
				record.ID(),
			)
		}
		index++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func assertOperationRecordsEqual(
	t *testing.T,
	want Package,
	got Package,
) {
	t.Helper()
	var records []Operation
	if err := got.VisitOperations(func(record Operation) error {
		records = append(records, record)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	index := 0
	if err := want.VisitOperations(func(record Operation) error {
		if index >= len(records) ||
			!equalOperationRecords(record, records[index]) {
			t.Fatalf("semantic operation %d differs", index)
		}
		index++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func assertUnsupportedRecordsEqual(
	t *testing.T,
	want Package,
	got Package,
) {
	t.Helper()
	var records []Unsupported
	if err := got.VisitUnsupported(func(record Unsupported) error {
		records = append(records, record)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	index := 0
	if err := want.VisitUnsupported(func(record Unsupported) error {
		if index >= len(records) ||
			!equalUnsupportedRecords(record, records[index]) ||
			record.Authority() != records[index].Authority() {
			t.Fatalf("semantic unsupported %d differs", index)
		}
		index++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
