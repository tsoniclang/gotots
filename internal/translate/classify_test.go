// The inventory site classifier is a TOTAL function over the CLOSED,
// producer-owned ir.UnsupportedKind enum: every kind maps to an
// explicitly reviewed disposition category, never a silent default. These
// pin that exhaustiveness (a newly declared kind with no reviewed
// disposition fails here), that classification is by Kind — so an
// ordered-prefix shadow can never misroute a site — and that the zero
// (never-set) kind is honestly "unclassified".
package translate

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/ir"
)

func TestDispositionByKindTotal(t *testing.T) {
	validCategories := map[string]bool{
		catLanguageLowering: true,
		catRepresentation:   true,
		catExternalContract: true,
		catProductPolicy:    true,
	}
	for _, kind := range ir.AllUnsupportedKinds() {
		category, root := classifySite(kind)
		if category == catUnclassified {
			t.Errorf("kind %q (%d) has no reviewed disposition; add it to dispositionByKind", kind, kind)
		}
		if !validCategories[category] {
			t.Errorf("kind %q maps to unknown category %q", kind, category)
		}
		if root == "" || root == rootUnclassifiedNote {
			t.Errorf("kind %q carries no reviewed root abstraction", kind)
		}
	}
}

func TestClassifySiteClassifiesStructuredConcurrency(t *testing.T) {
	// The concurrency statements classify under the single concurrency
	// product policy, resolved by their producer-owned kind.
	for _, kind := range []ir.UnsupportedKind{
		ir.KindGoroutineStatement, ir.KindSelectStatement, ir.KindChannelSendStatement,
	} {
		category, _ := classifySite(kind)
		if category != catProductPolicy {
			t.Errorf("concurrency kind %q classified as %q, want %q", kind, category, catProductPolicy)
		}
	}
}

func TestClassifySiteZeroKindIsUnclassified(t *testing.T) {
	// The zero (never-set) kind has no disposition — surfaced honestly as
	// unclassified rather than defaulted into a language lowering.
	category, _ := classifySite(ir.KindUnsupportedInvalid)
	if category != catUnclassified {
		t.Errorf("zero kind wrongly classified as %q", category)
	}
}
