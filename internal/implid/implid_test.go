package implid

import "testing"

func TestRoundTripWithSlashesInPackagePath(t *testing.T) {
	source := "github.com/microsoft/typescript-go/internal/tsoptions::func::ParseCommandLine"
	id := MustNew(source, "default")
	parsed, err := Parse(id.String())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Source != source || parsed.Key != "default" {
		t.Fatalf("round trip lost identity: %+v", parsed)
	}
}

func TestFamilyKeysRoundTrip(t *testing.T) {
	for _, key := range []string{"map-key-encoded", "pointer-cell"} {
		id := MustNew("pkg::method::T::M", key)
		parsed, err := Parse(id.String())
		if err != nil || parsed.Key != key {
			t.Fatalf("key %s: %v %+v", key, err, parsed)
		}
	}
}

// MUTATION: malformed inputs fail closed at both construction and parse.
func TestMalformedFailClosed(t *testing.T) {
	if _, err := New("no-canonical-identity", "default"); err == nil {
		t.Fatal("source without :: must fail")
	}
	if _, err := New("pkg::func::F", ""); err == nil {
		t.Fatal("empty key must fail")
	}
	if _, err := New("pkg::func::F", "a/b"); err == nil {
		t.Fatal("key with slash must fail")
	}
	for _, bad := range []string{"pkg::func::F", "github.com/x/y", "", "a/b/c"} {
		if _, err := Parse(bad); err == nil {
			t.Fatalf("Parse(%q) must fail", bad)
		}
	}
}
