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
	if firstRun == nil || firstRun.Shapes == nil || corpusGenerated == nil {
		return "blocked", []string{"census shapes or corpus generation did not run"}, nil
	}
	// The verified denominator is the PRODUCTION function/method shape
	// set: test-scope declarations receive their disposition through the
	// test-ledger stages, exactly like the census reconciliation.
	production := map[string]bool{}
	for _, decl := range firstRun.Report.Declarations {
		if decl.Scope == "production" {
			production[decl.ID] = true
		}
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
	if verified == 0 && len(defects) == 0 && denominator > unimplementedCount {
		return "fail", nil, fmt.Errorf("zero signatures verified: the proof set is empty or disjoint from the census")
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
	// Function literals: every census funclit shape must join to exactly
	// one translate disposition with the identical body hash — the
	// independent-unit ledger the specification requires.
	litStates := map[string]translate.FuncLitSupport{}
	for _, lit := range corpusGenerated.FuncLits {
		if _, dup := litStates[lit.ID]; dup {
			defects = append(defects, "duplicate funclit disposition "+lit.ID)
		}
		litStates[lit.ID] = lit
	}
	litJoined, litUnimplemented := 0, 0
	for _, shape := range firstRun.Shapes.FunctionLiterals {
		lit, has := litStates[shape.ID]
		if !has {
			defects = append(defects, "no funclit disposition for census shape "+shape.ID)
			continue
		}
		if shape.BodyHash == "" || lit.BodyHash == "" {
			// A function literal ALWAYS has a body: an empty hash is a
			// missing-evidence defect (an invalid source span), never a
			// silently-joined pair of empty strings.
			defects = append(defects, "funclit missing body-hash evidence at "+shape.ID)
			continue
		}
		if lit.BodyHash != shape.BodyHash {
			defects = append(defects, "funclit body-hash drift at "+shape.ID)
			continue
		}
		if lit.Parent != shape.Parent {
			defects = append(defects, "funclit parent drift at "+shape.ID+": census "+shape.Parent+" vs translate "+lit.Parent)
			continue
		}
		if lit.State == "unimplemented" {
			litUnimplemented++
		}
		litJoined++
		delete(litStates, shape.ID)
	}
	for id := range litStates {
		defects = append(defects, "orphan funclit disposition (no census shape): "+id)
	}
	if len(defects) > 0 {
		if len(defects) > 15 {
			defects = defects[:15]
		}
		return "fail", defects, fmt.Errorf("function-literal ledger failed the identity join")
	}
	// Variable initializers: each census initializer hash must join to
	// the var proof's recorded initializer hash.
	initHash := map[string]string{}
	for _, proof := range corpusGenerated.Proofs {
		if proof.InitHash != "" {
			initHash[proof.ID] = proof.InitHash
		}
	}
	varUnimplemented := map[string]bool{}
	for _, support := range corpusGenerated.Support {
		if string(support.State) == "unimplemented" {
			varUnimplemented[support.ID] = true
		}
	}
	initJoined, initBlocked := 0, 0
	for _, shape := range firstRun.Shapes.Variables {
		if shape.InitializerHash == "" || !production[shape.ID] {
			continue
		}
		if varUnimplemented[shape.ID] {
			// An explicitly unimplemented variable has its disposition;
			// hash evidence exists when the lowering does.
			initBlocked++
			continue
		}
		got, has := initHash[shape.ID]
		if !has {
			defects = append(defects, "no initializer evidence for census var "+shape.ID)
			continue
		}
		if got != shape.InitializerHash {
			defects = append(defects, "initializer hash drift at "+shape.ID)
			continue
		}
		initJoined++
	}
	if len(defects) > 0 {
		if len(defects) > 15 {
			defects = defects[:15]
		}
		return "fail", defects, fmt.Errorf("initializer evidence failed the identity join")
	}
	// Constants: every census constant shape must join to a translate const
	// proof carrying the IDENTICAL declaration-shape hash (canonical type
	// plus exact value), and no const proof may reference an identity absent
	// from the census.
	constHash := map[string]string{}
	for _, proof := range corpusGenerated.Proofs {
		if proof.ConstHash != "" {
			constHash[proof.ID] = proof.ConstHash
		}
	}
	constJoined := 0
	for _, shape := range firstRun.Shapes.Constants {
		if !production[shape.ID] {
			continue
		}
		expected := signatureHashOf(census.ConstShapeSignature(shape))
		got, has := constHash[shape.ID]
		if !has {
			defects = append(defects, "no constant evidence for census const "+shape.ID)
			continue
		}
		if got != expected {
			defects = append(defects, "constant shape drift at "+shape.ID)
			continue
		}
		constJoined++
	}
	censusConstIDs := map[string]bool{}
	for _, shape := range firstRun.Shapes.Constants {
		censusConstIDs[shape.ID] = true
	}
	for id := range constHash {
		if !censusConstIDs[id] {
			defects = append(defects, "orphan constant proof (no census shape): "+id)
		}
	}
	if len(defects) > 0 {
		if len(defects) > 15 {
			defects = defects[:15]
		}
		return "fail", defects, fmt.Errorf("constant shape ledger failed the identity join")
	}
	// What IS joined so far is real but PARTIAL, and it is agreement
	// between two callers of the SAME typeid.Canonical, not independent
	// structural verification of the emitted TypeScript against the Go
	// declaration. "Declaration/signature/type completeness" is therefore
	// not yet provable: this stage is BLOCKED until (a) every declaration
	// class joins and (b) a generated-TS shape extractor compares the
	// emitted declaration structurally with the Go declaration.
	return "blocked", []string{
		fmt.Sprintf("census production function/method denominator: %d", denominator),
		fmt.Sprintf("signatures joined against the census spelling: %d", verified),
		fmt.Sprintf("explicit unimplemented records: %d", unimplementedCount),
		fmt.Sprintf("orphan proofs: %d", orphans),
		fmt.Sprintf("function literals joined by identity, parent, and body hash: %d (%d unimplemented)", litJoined, litUnimplemented),
		fmt.Sprintf("variable initializers joined by identity and hash: %d (%d explicitly unimplemented)", initJoined, initBlocked),
		fmt.Sprintf("constants joined by canonical type and exact value: %d", constJoined),
		"BLOCKED: function signatures, function-literal bodies, variable initializers, and constants are joined — named types, variable TYPES, aliases, struct fields/tags/embedding, and interface methods/embeds are NOT yet verified (census records them in Shapes.* but this stage does not join them)",
		"BLOCKED: the join proves census and translator AGREE (both derive identity from internal/typeid.Canonical) — it is not INDEPENDENT correctness; no generated TypeScript declaration is structurally compared with its Go declaration",
	}, nil
}

// signatureHashOf is the canonical signature digest stage 05 verifies.
func signatureHashOf(signature string) string {
	digest := sha256.Sum256([]byte(signature))
	return hex.EncodeToString(digest[:])
}
