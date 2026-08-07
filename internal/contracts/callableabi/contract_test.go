package callableabi

import "testing"

func TestCallableProjectionIsClosedAndFingerprintIsStructural(t *testing.T) {
	identityKey, err := PackageFunctionIdentity("example.test/fast", "Read")
	if err != nil {
		t.Fatal(err)
	}
	pointee, err := NewParameter(
		ProjectionPointeeValue,
		NilPolicyRejectAtBoundary,
		"number",
	)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := New(identityKey, []Parameter{pointee})
	if err != nil {
		t.Fatal(err)
	}
	if !selected.Valid() || selected.Fingerprint() == "" || selected.Parameters()[0].Projection() != ProjectionPointeeValue {
		t.Fatalf("invalid callable projection: %#v", selected)
	}
	identity, err := NewParameter(
		ProjectionIdentity,
		NilPolicyNotApplicable,
		"GoPointer<number, number> | undefined",
	)
	if err != nil {
		t.Fatal(err)
	}
	other, err := New(identityKey, []Parameter{identity})
	if err != nil {
		t.Fatal(err)
	}
	if selected.Fingerprint() == other.Fingerprint() {
		t.Fatal("projection mutation did not change the callable fingerprint")
	}
	preserved, err := NewParameter(
		ProjectionPointeeValue,
		NilPolicyPreserve,
		"number | undefined",
	)
	if err != nil {
		t.Fatal(err)
	}
	preservedCallable, err := New(identityKey, []Parameter{preserved})
	if err != nil {
		t.Fatal(err)
	}
	if selected.Fingerprint() == preservedCallable.Fingerprint() {
		t.Fatal("nil-policy mutation did not change the callable fingerprint")
	}
}

func TestCallableIdentitiesDistinguishFunctionsAndMethods(t *testing.T) {
	function, err := PackageFunctionIdentity("example.test/source", "Read")
	if err != nil {
		t.Fatal(err)
	}
	method, err := MethodIdentity(
		"example.test/source",
		"*example.test/source.Reader",
		"Read",
	)
	if err != nil {
		t.Fatal(err)
	}
	if function == method {
		t.Fatal("package function and method identities collided")
	}
	if _, err := MethodIdentity("example.test/source", "", "Read"); err == nil {
		t.Fatal("method identity accepted an absent receiver")
	}
}
