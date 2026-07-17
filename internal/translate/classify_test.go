// The inventory site classifier is a TOTAL function over the closed
// construct-key set: every registered unsupported construct maps to an
// explicitly reviewed disposition category, never a silent default. This
// pins that exhaustiveness (a newly registered construct with no reviewed
// disposition fails here) and that classification is exact-match, not
// substring — an unregistered free-text construct is honestly
// "unclassified", not absorbed into a language lowering.
package translate

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/ir"
)

func TestClassifySiteIsTotalOverClosedConstructSet(t *testing.T) {
	validCategories := map[string]bool{
		catLanguageLowering: true,
		catRepresentation:   true,
		catExternalContract: true,
		catProductPolicy:    true,
	}
	for _, key := range ir.ClassConstructKeys() {
		// The class ir.ClassOf builds is "CODE: <construct>"; classify the
		// construct under a representative code.
		category, root := classifySite("GOTOTS_UNSUPPORTED_TYPE: " + key)
		if category == catUnclassified {
			t.Errorf("registered construct %q has no reviewed disposition; classify it explicitly", key)
		}
		if !validCategories[category] {
			t.Errorf("construct %q maps to unknown category %q", key, category)
		}
		if root == "" || root == rootUnclassifiedNote {
			t.Errorf("construct %q carries no reviewed root abstraction", key)
		}
	}
}

func TestClassifySiteClassifiesStructuredConcurrency(t *testing.T) {
	// The concurrency statements carry structured constructs (not raw %T
	// spellings) and classify under the single concurrency product policy.
	for _, construct := range []string{"goroutine statement", "select statement", "channel send statement"} {
		category, _ := classifySite("GOTOTS_UNSUPPORTED_STATEMENT: " + construct)
		if category != catProductPolicy {
			t.Errorf("concurrency construct %q classified as %q, want %q", construct, category, catProductPolicy)
		}
	}
}

func TestClassifySiteExactMatchNotSubstring(t *testing.T) {
	// A free-text construct with no registered prefix is unclassified —
	// never silently matched by a substring and defaulted.
	category, _ := classifySite("GOTOTS_UNSUPPORTED_STATEMENT: return arity mismatch")
	if category != catUnclassified {
		t.Errorf("unregistered construct wrongly classified as %q", category)
	}
}
