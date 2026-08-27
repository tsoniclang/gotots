package gostdlib

import (
	"strings"
	"testing"
)

func TestProviderBoundaryProfilesAreUniformlySynchronous(t *testing.T) {
	interfaces := providerBoundaryTestInterface(EffectSynchronous)
	for _, test := range []struct {
		name       string
		interfaces []ProviderCallableProfileInterfaceDocument
		callables  []ProviderCallableParameterDocument
	}{
		{name: "interface", interfaces: interfaces},
		{
			name: "callable",
			callables: []ProviderCallableParameterDocument{{
				Parameter: 1,
				Effect:    EffectSynchronous,
			}},
		},
		{
			name:       "interface-and-callable",
			interfaces: interfaces,
			callables: []ProviderCallableParameterDocument{{
				Parameter: 1,
				Effect:    EffectSynchronous,
			}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := providerProfileBoundaryEffect(
				test.interfaces,
				test.callables,
				"profile",
			)
			if err != nil || got != EffectSynchronous {
				t.Fatalf("boundary effect = %q, %v; want %q", got, err, EffectSynchronous)
			}
		})
	}

	if _, err := providerProfileBoundaryEffect(
		providerBoundaryTestInterface(EffectKind("async")),
		nil,
		"profile",
	); err == nil || !strings.Contains(err.Error(), "not synchronous") {
		t.Fatalf("non-synchronous interface error = %v", err)
	}
	if _, err := providerProfileBoundaryEffect(
		interfaces,
		[]ProviderCallableParameterDocument{{
			Parameter: 0,
			Effect:    EffectKind("awaitable"),
		}},
		"profile",
	); err == nil || !strings.Contains(err.Error(), "not synchronous") {
		t.Fatalf("non-synchronous callable error = %v", err)
	}

	key, err := BuildProviderCallableProfileKey(nil, []ProviderCallableProfileKeyCallable{{
		Parameter: 1,
		Effect:    EffectSynchronous,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if key == "" {
		t.Fatal("synchronous callback profile key is empty")
	}
	lookups := make(map[string]struct{})
	if err := recordProviderBoundaryProfile(
		lookups,
		"func:example",
		EffectSynchronous,
		"first",
	); err != nil {
		t.Fatal(err)
	}
	if err := recordProviderBoundaryProfile(
		lookups,
		"func:example",
		EffectSynchronous,
		"duplicate",
	); err == nil || !strings.Contains(err.Error(), "multiple profiles") {
		t.Fatalf("duplicate boundary error = %v", err)
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
