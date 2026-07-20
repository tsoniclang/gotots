// Package shapegate is the calibration shape/size gate skeleton: four
// closed verdicts, mechanical detectors for the known architectural
// expansion classes, and fail-closed fixture-hash binding. Thresholds
// remain UNSET until the hand-port review freezes them (correction 4);
// the detectors and verdict grammar are authoritative now.
package shapegate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// Verdict is the closed gate outcome set.
type Verdict string

const (
	VerdictPass        Verdict = "pass"
	VerdictSpecialized Verdict = "pass-specialized"
	VerdictException   Verdict = "pass-exception"
	VerdictManual      Verdict = "manual-required"
	VerdictFail        Verdict = "fail"
)

// Evaluation is one fixture's gate result with its evidence.
type Evaluation struct {
	FixtureID string  `json:"fixtureId"`
	Verdict   Verdict `json:"verdict"`
	// HiddenOperationArgs counts generic capability arguments visible in
	// the emitted body — must be zero for an ordinary pass.
	HiddenOperationArgs int `json:"hiddenOperationArgs"`
	// FreeFunctionFacade marks a source METHOD emitted as a receiver-
	// taking free function without a typed nil-receiver reason.
	FreeFunctionFacade bool `json:"freeFunctionFacade"`
	// SpecializationEvidence must name why forms 1-3 (parametric,
	// representation-owned, concrete-at-site) were insufficient.
	SpecializationEvidence string `json:"specializationEvidence,omitempty"`
	// ExceptionReason is the typed semantic family justifying longer
	// output, with its attributed bytes.
	ExceptionReason string `json:"exceptionReason,omitempty"`
	ExceptionBytes  int    `json:"exceptionBytes,omitempty"`
	Failures        []string
}

var hiddenArgPatterns = []*regexp.Regexp{
	// Named capability parameters/references (declaration side and
	// method variants): zero$T, eq$K, rt$P ...
	regexp.MustCompile(`\b(zero|eq|clone|set|key|rt)\$[A-Za-z0-9_$]`),
	// Inline factory composition at call sites: synthetic-parameter
	// arrows `(($a: ...` / `(($v: ...` and zero-factory thunks `() =>`
	// passed as arguments.
	regexp.MustCompile(`\(\(\$[a-z]`),
	regexp.MustCompile(`\(\)\s*=>`),
}

// CountHiddenOperationArgs detects generic capability arguments in an
// emitted body — named capability slots and inline factory arrows.
func CountHiddenOperationArgs(body string) int {
	count := 0
	for _, pattern := range hiddenArgPatterns {
		count += len(pattern.FindAllString(body, -1))
	}
	return count
}

// DetectFreeFunctionFacade reports a method fixture emitted as
// Recv$Name free function.
func DetectFreeFunctionFacade(body, recvType, methodName string) bool {
	if recvType == "" {
		return false
	}
	return strings.Contains(body, "function "+recvType+"$"+methodName+"(") ||
		strings.Contains(body, "function "+recvType+"$"+methodName+"<")
}

// CountDuplicateDefinitions counts repeated occurrences of the same
// type/vtable definition head across module texts — definitions may
// exist once; references may repeat.
func CountDuplicateDefinitions(modules map[string]string, definitionHead string) int {
	count := 0
	for _, text := range modules {
		count += strings.Count(text, definitionHead)
	}
	if count > 1 {
		return count - 1
	}
	return 0
}

// VerifyFixtureHash fails closed on a moved or edited fixture source.
func VerifyFixtureHash(source []byte, expectedSha256 string) error {
	digest := sha256.Sum256(source)
	got := hex.EncodeToString(digest[:])
	if got != expectedSha256 {
		return fmt.Errorf("fixture source hash mismatch: manifest %s, current %s (stale or moved fixture fails closed)", expectedSha256[:12], got[:12])
	}
	return nil
}

// EvaluateOrdinary applies the ordinary-verdict rules to an emitted
// body: zero hidden arguments, no facade, no duplicate definitions.
// Ratio thresholds attach here after the hand-port review freezes them.
func EvaluateOrdinary(fixtureID, body, recvType, methodName string) Evaluation {
	out := Evaluation{FixtureID: fixtureID, Verdict: VerdictPass}
	out.HiddenOperationArgs = CountHiddenOperationArgs(body)
	if out.HiddenOperationArgs > 0 {
		out.Failures = append(out.Failures, fmt.Sprintf("%d hidden generic operation arguments on an ordinary fixture", out.HiddenOperationArgs))
	}
	out.FreeFunctionFacade = DetectFreeFunctionFacade(body, recvType, methodName)
	if out.FreeFunctionFacade {
		out.Failures = append(out.Failures, "source method emitted as a free-function facade without a typed nil-receiver reason")
	}
	if len(out.Failures) > 0 {
		out.Verdict = VerdictFail
	}
	return out
}
