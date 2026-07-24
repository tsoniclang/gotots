package semantic

import (
	"runtime"
	"strings"
	"testing"
)

func TestAuthorityValidityIsConstructorOwned(t *testing.T) {
	if (Authority{}).Valid() {
		t.Fatal("zero authority is valid")
	}
	digest := strings.Repeat("ab", 32)
	checker, err := NewCheckerAuthority(
		digest,
		digest,
		digest,
		digest,
		digest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !checker.Valid() || checker.Kind() != AuthorityChecker {
		t.Fatal("validated checker authority is invalid")
	}
	provider, err := NewCertifiedProviderAuthority(
		digest,
		digest,
		digest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !provider.Valid() ||
		provider.Kind() != AuthorityCertifiedProvider {
		t.Fatal("validated provider authority is invalid")
	}
	for _, malformed := range []string{
		"",
		strings.Repeat("ab", 31),
		strings.Repeat("AB", 32),
		strings.Repeat("xz", 32),
	} {
		if _, err := NewCheckerAuthority(
			malformed,
			digest,
			digest,
			digest,
			digest,
		); err == nil {
			t.Fatalf("checker authority accepted digest %q", malformed)
		}
		if _, err := NewCertifiedProviderAuthority(
			malformed,
			digest,
			digest,
		); err == nil {
			t.Fatalf("provider authority accepted digest %q", malformed)
		}
	}
}

func TestAuthorityValidityIsAllocationFree(t *testing.T) {
	digest := strings.Repeat("ab", 32)
	authority, err := NewCheckerAuthority(
		digest,
		digest,
		digest,
		digest,
		digest,
	)
	if err != nil {
		t.Fatal(err)
	}
	valid := false
	allocations := testing.AllocsPerRun(1_000, func() {
		valid = authority.Valid()
	})
	runtime.KeepAlive(valid)
	if allocations != 0 {
		t.Fatalf(
			"authority validity allocates %.2f times per call",
			allocations,
		)
	}
	if !valid {
		t.Fatal("validated authority became invalid")
	}
}
