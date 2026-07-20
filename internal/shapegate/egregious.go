// Unconditional stop-the-line detectors: bounds so far outside any
// reviewable calibration threshold that they fail regardless of the
// pending independent review (which can only TIGHTEN them). The
// measured baseline cases — an ordinary 286 B body emitting 1,006 B
// (3.52x), a 424 B function emitting 59,612 B (140x), one alias
// definition copied into 101 modules, 3,897 inline box vtables — must
// each fail here. These are size/count gates; SHAPE authority belongs
// to the typed-AST verifier, never to text.
package shapegate

import (
	"fmt"
	"sort"
	"strings"
)

const (
	// EgregiousOrdinaryRatio is the unconditional generated/Go byte
	// bound for an ordinary-verdict fixture. Measured ordinary hand
	// ports sit near 1x; no review can accept 2.5x as ordinary.
	EgregiousOrdinaryRatio = 2.5
	// EgregiousAnyRatio is the unconditional bound for ANY fixture,
	// exception classes included.
	EgregiousAnyRatio = 20.0
	// EgregiousDefinitionCopies is the unconditional bound on emitted
	// copies of one definition identity: definitions are emitted once;
	// a second copy is already a stop-the-line duplication.
	EgregiousDefinitionCopies = 1
)

// StopTheLine is one unconditional failure.
type StopTheLine struct {
	Class string  `json:"class"`
	ID    string  `json:"id"`
	Value float64 `json:"value"`
	Bound float64 `json:"bound"`
}

func (s StopTheLine) String() string {
	return fmt.Sprintf("%s: %s value %.2f exceeds unconditional bound %.2f", s.Class, s.ID, s.Value, s.Bound)
}

// EgregiousExpansion evaluates one fixture's generated/Go byte ratio
// against the unconditional bounds.
func EgregiousExpansion(fixtureID, verdict string, goBytes, generatedBytes int) *StopTheLine {
	if goBytes <= 0 {
		return nil
	}
	ratio := float64(generatedBytes) / float64(goBytes)
	if verdict == "ordinary" && ratio > EgregiousOrdinaryRatio {
		return &StopTheLine{Class: "egregious-ordinary-expansion", ID: fixtureID, Value: ratio, Bound: EgregiousOrdinaryRatio}
	}
	if ratio > EgregiousAnyRatio {
		return &StopTheLine{Class: "egregious-expansion", ID: fixtureID, Value: ratio, Bound: EgregiousAnyRatio}
	}
	return nil
}

// EgregiousDefinitionDuplication evaluates emitted copies of one
// definition identity across the corpus (from a typed emission or
// alias ledger — counts, never content).
func EgregiousDefinitionDuplication(identity string, copies int) *StopTheLine {
	if copies > EgregiousDefinitionCopies {
		return &StopTheLine{Class: "egregious-definition-duplication", ID: identity,
			Value: float64(copies), Bound: float64(EgregiousDefinitionCopies)}
	}
	return nil
}

// EgregiousReport evaluates a full measurement set: per-fixture ratios
// plus per-identity definition copy counts, returning every failure
// sorted for determinism.
type FixtureMeasurement struct {
	FixtureID      string
	Verdict        string
	GoBytes        int
	GeneratedBytes int
}

func EgregiousReport(fixtures []FixtureMeasurement, definitionCopies map[string]int) []StopTheLine {
	var out []StopTheLine
	for _, fixture := range fixtures {
		if failure := EgregiousExpansion(fixture.FixtureID, fixture.Verdict, fixture.GoBytes, fixture.GeneratedBytes); failure != nil {
			out = append(out, *failure)
		}
	}
	identities := make([]string, 0, len(definitionCopies))
	for identity := range definitionCopies {
		identities = append(identities, identity)
	}
	sort.Strings(identities)
	for _, identity := range identities {
		if failure := EgregiousDefinitionDuplication(identity, definitionCopies[identity]); failure != nil {
			out = append(out, *failure)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Class != out[j].Class {
			return out[i].Class < out[j].Class
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// RenderStopTheLine renders failures for gate output.
func RenderStopTheLine(failures []StopTheLine) string {
	var b strings.Builder
	for _, failure := range failures {
		b.WriteString(failure.String())
		b.WriteString("\n")
	}
	return b.String()
}
