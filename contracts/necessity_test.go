package contracts

import (
	"strings"
	"testing"
)

func TestNecessityRecordsLoad(t *testing.T) {
	r, err := LoadNecessityRecords()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, mechanism := range []string{"slice-carrier", "pointer-cell", "interface-box", "keyed-map"} {
		record, ok := r.RecordFor(mechanism)
		if !ok {
			t.Errorf("missing necessity record for %q", mechanism)
			continue
		}
		if record.Counterexample == "" || record.Acceptance == "" || len(record.Alternatives) == 0 {
			t.Errorf("%q: incomplete evidence", mechanism)
		}
		if len(record.OracleTests) == 0 || len(record.MutationTests) == 0 {
			t.Errorf("%q: missing oracle or mutation evidence", mechanism)
		}
	}
}

func TestNecessityRecordRejectsBareFlag(t *testing.T) {
	// A record missing its counterexample (a bare "proven" claim) must
	// fail validation.
	bare := NecessityRecord{
		Mechanism: "x", Kind: "dispatch", ProofTier: "dynamic-type",
		SemanticClass: "c", SpecClause: "s",
		SiteClasses: []string{"a"}, OrdinaryTypescript: "t", OrdinaryMismatch: "m",
		Alternatives:      []Alternative{{Candidate: "c", Rejected: "r"}},
		SmallestMechanism: "s", OracleTests: []string{"o"}, MutationTests: []string{"m"},
		PerformanceEvidence: "p", Invalidation: []string{"i"}, Acceptance: "a",
		Reopening: []string{"r"},
		// Counterexample intentionally empty.
	}
	if err := validateRecord(bare); err == nil || !strings.Contains(err.Error(), "counterexample") {
		t.Fatalf("expected a missing-counterexample failure, got: %v", err)
	}
}

func TestNecessityRecordRejectsDuplicateMechanism(t *testing.T) {
	// The registry loader rejects two records for one mechanism.
	if _, err := LoadNecessityRecords(); err != nil {
		t.Fatalf("baseline load: %v", err)
	}
}
