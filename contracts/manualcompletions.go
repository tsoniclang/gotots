// The §C manual-completion contract: every typed-unimplemented body
// carries exactly one reviewed disposition — a product-policy exclusion
// (the body's feature is outside the runnable product) or an explicit
// deferred-protocol record (translator work remaining, publication
// honestly blocked). Nothing is silently out of scope: an unimplemented
// body without a disposition, a stale disposition without a body, or a
// body whose BLOCKERS drifted from the reviewed record all fail closed.
package contracts

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed manual-completions.json
var manualCompletionsJSON []byte

// Disposition is one reviewed verdict for one unimplemented body.
type Disposition struct {
	// Body is the census declaration ID.
	Body string `json:"body"`
	// Class routes the finding (concurrency, unsafe-memory,
	// platform-fswatch, runtime-recover, deferred-protocol).
	Class string `json:"class"`
	// Resolution is "accepted-manual" (a reviewed TS implementation
	// stands in), "product-policy" (excluded from the runnable
	// product), or "deferred" (translator protocol work remaining —
	// recorded, and publication completeness honestly blocked).
	Resolution string `json:"resolution"`
	// Constructs is the exact sorted set of blocker constructs the
	// review covered; a body acquiring different blockers invalidates
	// the disposition.
	Constructs []string `json:"constructs"`
	Note       string   `json:"note"`
}

// ManualCompletions is the loaded contract.
type ManualCompletions struct {
	SchemaVersion int           `json:"schemaVersion"`
	Dispositions  []Disposition `json:"dispositions"`
}

// LoadManualCompletions parses and structurally validates the embedded
// contract.
func LoadManualCompletions() (*ManualCompletions, error) {
	var out ManualCompletions
	if err := json.Unmarshal(manualCompletionsJSON, &out); err != nil {
		return nil, fmt.Errorf("manual-completions contract: %w", err)
	}
	if out.SchemaVersion != 1 {
		return nil, fmt.Errorf("manual-completions contract: unsupported schema %d", out.SchemaVersion)
	}
	seen := map[string]bool{}
	for _, d := range out.Dispositions {
		if d.Body == "" || d.Class == "" || d.Note == "" || len(d.Constructs) == 0 {
			return nil, fmt.Errorf("manual-completions contract: incomplete disposition for %q", d.Body)
		}
		switch d.Resolution {
		case "accepted-manual", "product-policy", "deferred":
		default:
			return nil, fmt.Errorf("manual-completions contract: unknown resolution %q for %s", d.Resolution, d.Body)
		}
		if seen[d.Body] {
			return nil, fmt.Errorf("manual-completions contract: duplicate disposition for %s", d.Body)
		}
		seen[d.Body] = true
	}
	return &out, nil
}

// VerifyDispositions checks the contract against the corpus's actual
// unimplemented bodies (body ID -> sorted distinct blocker constructs):
// every body needs exactly one matching disposition, every disposition
// needs its body, and the blocker sets must match the reviewed record.
func (m *ManualCompletions) VerifyDispositions(unimplemented map[string][]string) error {
	byBody := map[string]Disposition{}
	for _, d := range m.Dispositions {
		byBody[d.Body] = d
	}
	var defects []string
	bodies := make([]string, 0, len(unimplemented))
	for body := range unimplemented {
		bodies = append(bodies, body)
	}
	sort.Strings(bodies)
	for _, body := range bodies {
		disposition, has := byBody[body]
		if !has {
			defects = append(defects, "unimplemented body without a reviewed disposition: "+body)
			continue
		}
		want := append([]string(nil), disposition.Constructs...)
		sort.Strings(want)
		got := append([]string(nil), unimplemented[body]...)
		sort.Strings(got)
		if strings.Join(want, "\x00") != strings.Join(got, "\x00") {
			defects = append(defects, fmt.Sprintf("disposition drift at %s: reviewed blockers %v, actual %v", body, want, got))
		}
		delete(byBody, body)
	}
	stale := make([]string, 0, len(byBody))
	for body := range byBody {
		stale = append(stale, body)
	}
	sort.Strings(stale)
	for _, body := range stale {
		defects = append(defects, "stale disposition without an unimplemented body: "+body)
	}
	if len(defects) > 0 {
		return fmt.Errorf("manual-completion contract violations:\n%s", strings.Join(defects, "\n"))
	}
	return nil
}
