package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/contracts"
	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/staticness"
	"github.com/tsoniclang/gotots/internal/translate"
)

// runRepresentationGate is acceptance stage 08: every representation
// entry classifies into the closed registry, every census-derived
// obligation (receiver, parameters, results) has carrier evidence, and
// every emitted custom mechanism carries a verified necessity record.
func runRepresentationGate(repoDir string, firstRun *census.Result, corpusGenerated *translate.Generated) (string, []string, error) {
	if corpusGenerated == nil {
		return "blocked", []string{"corpus generation did not run"}, nil
	}
	// The slice family's fixed point is live: every generated body's
	// proof records its regions' selections, and the deterministic
	// re-run inside this gate's second generation (stage 09) already
	// proved plan stability. Here the selections themselves are
	// verified: a native selection may appear only with the direct
	// lowering evidence, never alongside carrier operations.
	//
	// Interface payload erasure is NOT re-checked textually here: the AST
	// staticness verifier (VerifyAST, below) owns that policy structurally
	// — flagging any GoBox use whose payload is an erased top (any, unknown,
	// object, {}, an alias to any of those, or an unconstrained generic
	// parameter) and any recovery cast off a box's .v — so a substring scan
	// of goiface.ts would only duplicate it less completely.
	// EVERY representation entry in EVERY family must classify into
	// the versioned registry's closed candidate sets — an unknown key
	// family or unknown candidate anywhere is forged or drifted
	// evidence, never only for slices.
	planned := map[string]int{}
	familyCounts := map[string]int{}
	var invalid []string
	for _, proof := range corpusGenerated.Proofs {
		for key, value := range proof.Representations {
			family, ok := translate.ClassifyRepresentation(key, value)
			if !ok {
				if len(invalid) < 10 {
					invalid = append(invalid, fmt.Sprintf("%s %s: unknown candidate %q", proof.ID, key, value))
				}
				continue
			}
			familyCounts[family]++
			if family == "slice-local" {
				planned[value]++
			}
		}
	}
	if len(invalid) > 0 {
		return "fail", invalid, fmt.Errorf("representation evidence outside the closed registry")
	}
	// Deletion detection: the OBLIGATION set is derived independently
	// from census shapes (receiver, params, and results share the
	// translator's canonical identity), and every obligation must
	// appear in the proof's carrier evidence.
	proofCarriers := map[string]map[string]bool{}
	for _, proof := range corpusGenerated.Proofs {
		keys := map[string]bool{}
		for key := range proof.Representations {
			keys[key] = true
		}
		proofCarriers[proof.ID] = keys
	}
	var missingEvidence []string
	for _, shape := range firstRun.Shapes.Functions {
		keys, has := proofCarriers[shape.ID]
		if !has {
			continue // unimplemented bodies carry no proof (stage 05 joins them)
		}
		// The receiver is a carrier like any parameter: its evidence is
		// obligated so deleting the receiver's carrier record fails.
		if shape.Receiver != "" && !keys["carrier:"+shape.Receiver] {
			if len(missingEvidence) < 10 {
				missingEvidence = append(missingEvidence, shape.ID+": missing carrier evidence for receiver "+shape.Receiver)
			}
		}
		for _, param := range shape.Params {
			if !keys["carrier:"+param.Type] {
				if len(missingEvidence) < 10 {
					missingEvidence = append(missingEvidence, shape.ID+": missing carrier evidence for "+param.Type)
				}
			}
		}
		for _, result := range shape.Results {
			if !keys["carrier:"+result.Type] {
				if len(missingEvidence) < 10 {
					missingEvidence = append(missingEvidence, shape.ID+": missing carrier evidence for "+result.Type)
				}
			}
		}
	}
	if len(missingEvidence) > 0 {
		return "fail", missingEvidence, fmt.Errorf("representation obligations missing from proof evidence")
	}
	if planned["native-array"]+planned["goslice-carrier"] == 0 {
		return "blocked", []string{"no slice regions recorded; planner evidence missing"}, nil
	}
	// Every custom runtime mechanism present in generated output must
	// have a machine-checked necessity record (spec 10): the record
	// carries the counterexample, ordinary-TypeScript mismatch,
	// rejected alternatives, oracle/mutation/perf evidence,
	// invalidation, acceptance, and reopening — a bare "proven" flag
	// is rejected. The requirement is derived structurally from the
	// emitted output, not a generator flag.
	necessity, err := contracts.LoadNecessityRecords()
	if err != nil {
		return "fail", nil, err
	}
	// Mechanisms are derived from the typed AST of the generated
	// output (resolved identifiers, not substrings), and every
	// record's oracle and mutation evidence must resolve to actual
	// committed test functions — prose is not evidence.
	typescriptModule := filepath.Join(repoDir, "product", "node_modules", "typescript")
	astReport, err := staticness.VerifyAST(corpusGenerated.Files, typescriptModule)
	if err != nil {
		return "fail", nil, err
	}
	present := make([]string, 0, len(astReport.Mechanisms))
	for mechanism := range astReport.Mechanisms {
		present = append(present, mechanism)
	}
	sort.Strings(present)
	var problems []string
	testIndex, err := committedTestFunctions(repoDir)
	if err != nil {
		return "fail", nil, err
	}
	for _, mechanism := range present {
		record, ok := necessity.RecordFor(mechanism)
		if !ok {
			problems = append(problems, mechanism+": no necessity record")
			continue
		}
		for _, test := range record.OracleTests {
			if !testIndex[test] {
				problems = append(problems, fmt.Sprintf("%s: oracle evidence %q is not a committed test function", mechanism, test))
			}
		}
		for _, test := range record.MutationTests {
			if !testIndex[test] {
				problems = append(problems, fmt.Sprintf("%s: mutation evidence %q is not a committed test function", mechanism, test))
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return "fail", problems, fmt.Errorf("%d necessity-record defects", len(problems))
	}
	familyDetail := make([]string, 0, len(familyCounts))
	for family, count := range familyCounts {
		familyDetail = append(familyDetail, fmt.Sprintf("%s:%d", family, count))
	}
	sort.Strings(familyDetail)
	details := []string{
		fmt.Sprintf("slice regions planned within the closed candidate set: native-array %d, goslice-carrier %d",
			planned["native-array"], planned["goslice-carrier"]),
		fmt.Sprintf("representation entries per family (all registry-classified): %v", familyDetail),
		fmt.Sprintf("custom mechanisms with verified necessity records (AST-derived): %v", present),
	}
	// Bind the necessity evidence to EXECUTED results: run the exact
	// named oracle and mutation test functions and require every one to
	// pass — prose or unexecuted names are not evidence.
	evidenceTests := map[string]bool{}
	for _, mechanism := range present {
		record, _ := necessity.RecordFor(mechanism)
		for _, test := range record.OracleTests {
			evidenceTests[test] = true
		}
		for _, test := range record.MutationTests {
			evidenceTests[test] = true
		}
	}
	names := make([]string, 0, len(evidenceTests))
	for name := range evidenceTests {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "fail", details, fmt.Errorf("no necessity evidence tests to execute")
	}
	pattern := "^(" + strings.Join(names, "|") + ")$"
	output, testErr := runInRepo(repoDir, "go", "test", "-count=1", "-json", "-run", pattern, "./...")
	passed := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		var event struct {
			Action string `json:"Action"`
			Test   string `json:"Test"`
		}
		if json.Unmarshal([]byte(line), &event) != nil {
			continue
		}
		if event.Action == "pass" && event.Test != "" {
			passed[event.Test] = true
		}
	}
	var unexecuted []string
	for _, name := range names {
		if !passed[name] {
			unexecuted = append(unexecuted, name)
		}
	}
	if testErr != nil || len(unexecuted) > 0 {
		if len(unexecuted) > 10 {
			unexecuted = unexecuted[:10]
		}
		details = append(details, "necessity evidence not green when executed: "+strings.Join(unexecuted, ", "))
		return "fail", details, fmt.Errorf("necessity evidence execution failed (%d tests not passing)", len(unexecuted))
	}
	details = append(details,
		fmt.Sprintf("necessity evidence EXECUTED: %d named test functions green across %d mechanisms", len(names), len(present)))
	return "pass", details, nil
}
