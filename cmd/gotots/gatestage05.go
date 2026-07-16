// Acceptance stage 05: declaration/signature/type completeness verified
// independently over the census denominator.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/translate"
)

// runSignatureCompletenessGate is acceptance stage 05: every census
// function/method shape is independently verified against its proof
// signature or an explicit unimplemented record, over the census
// denominator, with orphan proofs rejected.
func runSignatureCompletenessGate(firstRun *census.Result, corpusGenerated *translate.Generated) (string, []string, error) {
	// The verified denominator is the PRODUCTION function/method shape
	// set: test-scope declarations receive their disposition through the
	// test-ledger stages, exactly like the census reconciliation.
	production := map[string]bool{}
	for _, decl := range firstRun.Report.Declarations {
		if decl.Scope == "production" {
			production[decl.ID] = true
		}
	}

	if firstRun == nil || firstRun.Shapes == nil || corpusGenerated == nil {
		return "blocked", []string{"census shapes or corpus generation did not run"}, nil
	}
	// Independent verification over the CENSUS DENOMINATOR, never
	// only the proofs that happen to exist: every census function and
	// method shape must be accounted for — a matching proof signature
	// hash, or an explicit unimplemented support record. A missing
	// proof is a detected defect; an empty proof set cannot pass.
	proofHash := map[string]string{}
	for _, proof := range corpusGenerated.Proofs {
		if proof.SignatureHash != "" {
			proofHash[proof.ID] = proof.SignatureHash
		}
	}
	supportState := map[string]string{}
	for _, support := range corpusGenerated.Support {
		supportState[support.ID] = string(support.State)
	}
	verified, unimplementedCount, denominator := 0, 0, 0
	var defects []string
	for _, shape := range firstRun.Shapes.Functions {
		if !production[shape.ID] {
			continue
		}
		denominator++
		digest := sha256.Sum256([]byte(shape.Signature))
		expected := hex.EncodeToString(digest[:])
		if hash, has := proofHash[shape.ID]; has {
			if hash != expected {
				if len(defects) < 10 {
					defects = append(defects, "signature drift at "+shape.ID)
				} else {
					defects = append(defects, "")
				}
				continue
			}
			verified++
			continue
		}
		if state, has := supportState[shape.ID]; has && state == "unimplemented" {
			unimplementedCount++
			continue
		}
		if len(defects) < 10 {
			defects = append(defects, "no proof and no unimplemented record for census shape "+shape.ID)
		} else {
			defects = append(defects, "")
		}
	}
	if verified == 0 {
		return "fail", nil, fmt.Errorf("zero signatures verified: the proof set is empty or disjoint from the census")
	}
	// Reverse join: a proof for an identity absent from the census is
	// an orphan record.
	censusIDs := map[string]bool{}
	for _, shape := range firstRun.Shapes.Functions {
		censusIDs[shape.ID] = true
	}
	// (Orphan detection keeps the full shape universe: a proof matching a
	// test-scope shape is still identity-consistent.)
	orphans := 0
	for id := range proofHash {
		if !censusIDs[id] {
			orphans++
			if len(defects) < 10 {
				defects = append(defects, "orphan proof (no census shape): "+id)
			}
		}
	}
	if n := len(defects); n > 0 {
		compact := defects[:0]
		for _, d := range defects {
			if d != "" {
				compact = append(compact, d)
			}
		}
		return "fail", compact, fmt.Errorf("%d census identities failed independent signature verification", n)
	}
	return "pass", []string{
		fmt.Sprintf("census production function/method denominator: %d", denominator),
		fmt.Sprintf("signatures independently verified against the census spelling: %d", verified),
		fmt.Sprintf("explicit unimplemented records: %d", unimplementedCount),
		fmt.Sprintf("orphan proofs: %d", orphans),
	}, nil
}
