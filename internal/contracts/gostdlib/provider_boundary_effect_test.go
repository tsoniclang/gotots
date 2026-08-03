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
		callables  []ProviderCallableParameterDocument
		want       EffectKind
	}{
		{name: "direct", interfaces: direct, want: EffectSynchronous},
		{name: "cooperative", interfaces: cooperative, want: EffectAwaitable},
		{
			name: "callback-only",
			callables: []ProviderCallableParameterDocument{{
				Parameter: 1,
				Effect:    EffectAwaitable,
			}},
			want: EffectAwaitable,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := providerProfileBoundaryEffect(
				test.interfaces,
				test.callables,
				"profile",
			)
			if err != nil || got != test.want {
				t.Fatalf("boundary effect = %q, %v; want %q", got, err, test.want)
			}
		})
	}

	mixed := append(
		providerBoundaryTestInterface(EffectSynchronous),
		providerBoundaryTestInterface(EffectAwaitable)...,
	)
	if _, err := providerProfileBoundaryEffect(mixed, nil, "profile"); err == nil ||
		!strings.Contains(err.Error(), "mixes direct and cooperative") {
		t.Fatalf("mixed boundary error = %v", err)
	}
	if _, err := providerProfileBoundaryEffect(
		providerBoundaryTestInterface(EffectAsynchronous),
		nil,
		"profile",
	); err == nil || !strings.Contains(err.Error(), "neither direct nor awaitable") {
		t.Fatalf("asynchronous boundary error = %v", err)
	}
	if _, err := providerProfileBoundaryEffect(
		direct,
		[]ProviderCallableParameterDocument{{
			Parameter: 0,
			Effect:    EffectAwaitable,
		}},
		"profile",
	); err == nil || !strings.Contains(err.Error(), "mixes direct and cooperative") {
		t.Fatalf("mixed callback boundary error = %v", err)
	}

	directKey, err := BuildProviderCallableProfileKey(nil, []ProviderCallableProfileKeyCallable{{
		Parameter: 1,
		Effect:    EffectSynchronous,
	}})
	if err != nil {
		t.Fatal(err)
	}
	cooperativeKey, err := BuildProviderCallableProfileKey(
		nil,
		[]ProviderCallableProfileKeyCallable{{
			Parameter: 1,
			Effect:    EffectAwaitable,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if directKey == cooperativeKey {
		t.Fatal("callback effects produced the same profile key")
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
