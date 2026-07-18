// The method-slot / method-key emission choke point fails closed: an empty
// canonical identity can never silently become a bare-name substitution.
package emit

import "testing"

func TestRequireIdentityPanicsOnEmpty(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("requireIdentity must panic on an empty identity, not substitute a name")
		}
	}()
	_ = requireIdentity("", "some method")
}

func TestRequireIdentityPassesNonEmpty(t *testing.T) {
	if got := requireIdentity("Convert|abc", "m"); got != "Convert|abc" {
		t.Fatalf("requireIdentity returned %q", got)
	}
}
