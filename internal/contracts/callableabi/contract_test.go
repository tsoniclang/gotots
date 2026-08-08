package callableabi

import "testing"

func TestCallableParameterContractIsIdentityPreservingAndStructural(t *testing.T) {
	identityKey, err := PackageFunctionIdentity("example.test/fast", "Read")
	if err != nil {
		t.Fatal(err)
	}
	pointer, err := NewParameter("Pointer<number> | undefined")
	if err != nil {
		t.Fatal(err)
	}
	selected, err := New(identityKey, []Parameter{pointer}, "number")
	if err != nil {
		t.Fatal(err)
	}
	if !selected.Valid() || selected.Fingerprint() == "" || selected.Parameters()[0].Projection() != ProjectionIdentity {
		t.Fatalf("invalid callable parameter contract: %#v", selected)
	}
	changed, err := NewParameter("Pointer<bigint> | undefined")
	if err != nil {
		t.Fatal(err)
	}
	other, err := New(identityKey, []Parameter{changed}, "number")
	if err != nil {
		t.Fatal(err)
	}
	if selected.Fingerprint() == other.Fingerprint() {
		t.Fatal("target-type mutation did not change the callable fingerprint")
	}
	changedResult, err := New(identityKey, []Parameter{pointer}, "string")
	if err != nil {
		t.Fatal(err)
	}
	if selected.Fingerprint() == changedResult.Fingerprint() {
		t.Fatal("result-type mutation did not change the callable fingerprint")
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
