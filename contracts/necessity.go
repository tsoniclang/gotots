// Necessity records (docs/spec/10-machine-contracts-diagnostics.md): one
// machine-checked record per custom runtime mechanism — a representation
// carrier, indirection cell, dispatch, or boundary adapter — each with a
// minimized counterexample, the ordinary-TypeScript mismatch, the
// rejected alternatives, the smallest mechanism, oracle/mutation/perf
// evidence, invalidation dependencies, acceptance, and reopening
// conditions. The generator cannot set a bare "proven" flag: the gate
// joins each emitted mechanism to its record structurally.
package contracts

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed necessity-records.json
var necessityRecordsJSON []byte

// Alternative is one rejected representation candidate.
type Alternative struct {
	Candidate string `json:"candidate"`
	Rejected  string `json:"rejected"`
}

// NecessityRecord is one custom-mechanism necessity contract.
type NecessityRecord struct {
	Mechanism           string        `json:"mechanism"`
	Kind                string        `json:"kind"`
	ProofTier           string        `json:"proofTier"`
	SemanticClass       string        `json:"semanticClass"`
	SpecClause          string        `json:"specClause"`
	SiteClasses         []string      `json:"siteClasses"`
	Counterexample      string        `json:"counterexample"`
	OrdinaryTypescript  string        `json:"ordinaryTypescript"`
	OrdinaryMismatch    string        `json:"ordinaryMismatch"`
	Alternatives        []Alternative `json:"alternatives"`
	SmallestMechanism   string        `json:"smallestMechanism"`
	OracleTests         []string      `json:"oracleTests"`
	MutationTests       []string      `json:"mutationTests"`
	PerformanceEvidence string        `json:"performanceEvidence"`
	Invalidation        []string      `json:"invalidation"`
	Acceptance          string        `json:"acceptance"`
	Reopening           []string      `json:"reopening"`
}

// NecessityRegistry is the parsed necessity-record set.
type NecessityRegistry struct {
	SchemaVersion int               `json:"schemaVersion"`
	Records       []NecessityRecord `json:"records"`

	byMechanism map[string]NecessityRecord
}

// LoadNecessityRecords parses the embedded necessity records fail-closed.
func LoadNecessityRecords() (*NecessityRegistry, error) {
	var registry NecessityRegistry
	decoder := json.NewDecoder(bytes.NewReader(necessityRecordsJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&registry); err != nil {
		return nil, fmt.Errorf("parse necessity records: %w", err)
	}
	if decoder.More() {
		return nil, fmt.Errorf("necessity records: trailing content")
	}
	if registry.SchemaVersion != 1 {
		return nil, fmt.Errorf("necessity records: unsupported schemaVersion %d", registry.SchemaVersion)
	}
	registry.byMechanism = map[string]NecessityRecord{}
	for _, record := range registry.Records {
		if _, dup := registry.byMechanism[record.Mechanism]; dup {
			return nil, fmt.Errorf("necessity records: duplicate mechanism %q", record.Mechanism)
		}
		if err := validateRecord(record); err != nil {
			return nil, err
		}
		registry.byMechanism[record.Mechanism] = record
	}
	return &registry, nil
}

// validateRecord fails closed on a record missing any required evidence:
// a bare acceptance without a counterexample, mismatch, alternative,
// oracle, mutation, invalidation, or reopening condition is not a proof.
func validateRecord(record NecessityRecord) error {
	missing := func(name string, empty bool) error {
		if empty {
			return fmt.Errorf("necessity record %q: missing %s", record.Mechanism, name)
		}
		return nil
	}
	for _, check := range []struct {
		name  string
		empty bool
	}{
		{"counterexample", record.Counterexample == ""},
		{"ordinaryMismatch", record.OrdinaryMismatch == ""},
		{"smallestMechanism", record.SmallestMechanism == ""},
		{"acceptance", record.Acceptance == ""},
		{"siteClasses", len(record.SiteClasses) == 0},
		{"alternatives", len(record.Alternatives) == 0},
		{"oracleTests", len(record.OracleTests) == 0},
		{"mutationTests", len(record.MutationTests) == 0},
		{"invalidation", len(record.Invalidation) == 0},
		{"reopening", len(record.Reopening) == 0},
	} {
		if err := missing(check.name, check.empty); err != nil {
			return err
		}
	}
	return nil
}

// RecordFor returns the necessity record covering a mechanism, if any.
func (r *NecessityRegistry) RecordFor(mechanism string) (NecessityRecord, bool) {
	record, ok := r.byMechanism[mechanism]
	return record, ok
}

// Mechanisms lists the recorded mechanism identities in sorted order.
func (r *NecessityRegistry) Mechanisms() []string {
	out := make([]string, 0, len(r.byMechanism))
	for mechanism := range r.byMechanism {
		out = append(out, mechanism)
	}
	sort.Strings(out)
	return out
}
