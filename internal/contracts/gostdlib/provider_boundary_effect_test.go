package gostdlib

import (
	"strings"
	"testing"
)

func TestProviderBoundaryProfilesAreUniformAndModeBounded(t *testing.T) {
	direct := providerBoundaryTestInterface(EffectSynchronous)
	cooperative := providerBoundaryTestInterface(EffectAwaitable)

	for _, test := range []struct {
		name       string
		interfaces []ProviderCallableProfileInterfaceDocument
		want       EffectKind
	}{
		{name: "direct", interfaces: direct, want: EffectSynchronous},
		{name: "cooperative", interfaces: cooperative, want: EffectAwaitable},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := providerProfileBoundaryEffect(test.interfaces, "profile")
			if err != nil || got != test.want {
				t.Fatalf("boundary effect = %q, %v; want %q", got, err, test.want)
			}
		})
	}

	mixed := append(
		providerBoundaryTestInterface(EffectSynchronous),
		providerBoundaryTestInterface(EffectAwaitable)...,
	)
	if _, err := providerProfileBoundaryEffect(mixed, "profile"); err == nil ||
		!strings.Contains(err.Error(), "mixes direct and cooperative") {
		t.Fatalf("mixed boundary error = %v", err)
	}
	if _, err := providerProfileBoundaryEffect(
		providerBoundaryTestInterface(EffectAsynchronous),
		"profile",
	); err == nil || !strings.Contains(err.Error(), "neither direct nor awaitable") {
		t.Fatalf("asynchronous boundary error = %v", err)
	}

	lookups := make(map[string]struct{})
	if err := recordProviderBoundaryProfile(
		lookups,
		"func:example",
		EffectSynchronous,
		"direct",
	); err != nil {
		t.Fatal(err)
	}
	if err := recordProviderBoundaryProfile(
		lookups,
		"func:example",
		EffectAwaitable,
		"cooperative",
	); err != nil {
		t.Fatal(err)
	}
	if err := recordProviderBoundaryProfile(
		lookups,
		"func:example",
		EffectAwaitable,
		"duplicate",
	); err == nil || !strings.Contains(err.Error(), "multiple profiles") {
		t.Fatalf("duplicate boundary-mode error = %v", err)
	}
}

func providerBoundaryTestInterface(
	effect EffectKind,
) []ProviderCallableProfileInterfaceDocument {
	return []ProviderCallableProfileInterfaceDocument{{
		ProviderInterface: ProviderInterfaceDocument{
			Methods: []ProviderInterfaceMethodDocument{{Effect: effect}},
		},
	}}
}
