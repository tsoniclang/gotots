// Acceptance stage 05: declaration/signature/type completeness verified
// independently over the census denominator.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/translate"
)

// runSignatureCompletenessGate is acceptance stage 05: every census
// function/method shape is independently verified against its proof
// signature or an explicit unimplemented record, over the census
// denominator, with orphan proofs rejected.
func runSignatureCompletenessGate(firstRun *census.Result, corpusGenerated *translate.Generated, repoDir string) (string, []string, error) {
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
	// Named types: every census type shape must join to a translate proof
	// carrying the IDENTICAL complete-shape hash (kind, underlying, type
	// params, fields/tags/embeds, interface methods/embeds, declared
	// methods), or hold an explicit unimplemented record.
	typeShapeHash := map[string]string{}
	aliasShapeHash := map[string]string{}
	varTypeHash := map[string]string{}
	for _, proof := range corpusGenerated.Proofs {
		if proof.TypeShapeHash != "" {
			typeShapeHash[proof.ID] = proof.TypeShapeHash
		}
		if proof.AliasShapeHash != "" {
			aliasShapeHash[proof.ID] = proof.AliasShapeHash
		}
		if proof.VarTypeHash != "" {
			varTypeHash[proof.ID] = proof.VarTypeHash
		}
	}
	typeJoined, typeUnimplemented := 0, 0
	for _, shape := range firstRun.Shapes.Types {
		if !production[shape.ID] {
			continue
		}
		got, has := typeShapeHash[shape.ID]
		if !has {
			if state, hasState := supportState[shape.ID]; hasState && state == "unimplemented" {
				typeUnimplemented++
				continue
			}
			defects = append(defects, "no type-shape evidence for census type "+shape.ID)
			continue
		}
		if got != signatureHashOf(census.TypeShapeSignature(shape)) {
			defects = append(defects, "type shape drift at "+shape.ID)
			continue
		}
		typeJoined++
	}
	for id := range typeShapeHash {
		if !censusIDs[id] && !censusTypeID(firstRun, id) {
			defects = append(defects, "orphan type-shape proof (no census shape): "+id)
		}
	}
	// Aliases: identical target + own-type-parameter shape.
	aliasJoined, aliasUnimplemented := 0, 0
	for _, shape := range firstRun.Shapes.Aliases {
		if !production[shape.ID] {
			continue
		}
		got, has := aliasShapeHash[shape.ID]
		if !has {
			if state, hasState := supportState[shape.ID]; hasState && state == "unimplemented" {
				aliasUnimplemented++
				continue
			}
			defects = append(defects, "no alias-shape evidence for census alias "+shape.ID)
			continue
		}
		if got != signatureHashOf(census.AliasShapeSignature(shape)) {
			defects = append(defects, "alias shape drift at "+shape.ID)
			continue
		}
		aliasJoined++
	}
	// Variable TYPES: identical canonical type spelling (the initializer
	// join above already covers initializer identity).
	varTypeJoined, varTypeUnimplemented := 0, 0
	for _, shape := range firstRun.Shapes.Variables {
		if !production[shape.ID] {
			continue
		}
		got, has := varTypeHash[shape.ID]
		if !has {
			if varUnimplemented[shape.ID] {
				varTypeUnimplemented++
				continue
			}
			// Blank variables carry positional identities and no declared
			// binding; their effect-only/no-output proofs dispose them in
			// the census reconciliation stage.
			if strings.Contains(shape.ID, "::var::_@") {
				continue
			}
			defects = append(defects, "no variable-type evidence for census var "+shape.ID)
			continue
		}
		if got != signatureHashOf(shape.Type) {
			defects = append(defects, "variable type drift at "+shape.ID)
			continue
		}
		varTypeJoined++
	}
	if len(defects) > 0 {
		if len(defects) > 15 {
			defects = defects[:15]
		}
		return "fail", defects, fmt.Errorf("type/alias/variable declaration shapes failed the identity join")
	}
	// INDEPENDENCE: the emitted modules are parsed with the pinned
	// compiler and every module-retained declaration's structure is
	// compared with its census Go shape — never through the shared
	// canonical renderer.
	parityVerified, parityDefects, err := runDeclParityCheck(firstRun, corpusGenerated, repoDir)
	if err != nil {
		return "fail", nil, err
	}
	if len(parityDefects) > 0 {
		return "fail", parityDefects, fmt.Errorf("%d declarations failed independent structural parity with their Go shapes", len(parityDefects))
	}
	if parityVerified == 0 {
		return "fail", nil, fmt.Errorf("independent structural parity verified zero declarations: the parsed surface is empty or disjoint")
	}
	return "pass", []string{
		fmt.Sprintf("census production function/method denominator: %d", denominator),
		fmt.Sprintf("signatures joined against the census spelling: %d", verified),
		fmt.Sprintf("explicit unimplemented records: %d", unimplementedCount),
		fmt.Sprintf("orphan proofs: %d", orphans),
		fmt.Sprintf("function literals joined by identity, parent, and body hash: %d (%d unimplemented)", litJoined, litUnimplemented),
		fmt.Sprintf("variable initializers joined by identity and hash: %d (%d explicitly unimplemented)", initJoined, initBlocked),
		fmt.Sprintf("constants joined by canonical type and exact value: %d", constJoined),
		fmt.Sprintf("named types joined by complete shape (fields/tags/embeds/methods): %d (%d explicitly unimplemented)", typeJoined, typeUnimplemented),
		fmt.Sprintf("aliases joined by target and type parameters: %d (%d explicitly unimplemented)", aliasJoined, aliasUnimplemented),
		fmt.Sprintf("variable declared types joined: %d (%d explicitly unimplemented)", varTypeJoined, varTypeUnimplemented),
		fmt.Sprintf("independent structural parity (pinned-compiler parse vs Go shapes): %d declarations verified", parityVerified),
	}, nil
}

// censusTypeID reports whether an identity names a census type or alias
// shape.
func censusTypeID(firstRun *census.Result, id string) bool {
	for _, shape := range firstRun.Shapes.Types {
		if shape.ID == id {
			return true
		}
	}
	for _, shape := range firstRun.Shapes.Aliases {
		if shape.ID == id {
			return true
		}
	}
	return false
}

// signatureHashOf is the canonical signature digest stage 05 verifies.
func signatureHashOf(signature string) string {
	digest := sha256.Sum256([]byte(signature))
	return hex.EncodeToString(digest[:])
}
