// The versioned representation registry: the closed vocabulary of every
// candidate a proof may record, per semantic family. Proof records join
// to this registry at the acceptance gate — an unknown key family or an
// unknown candidate in ANY family is forged or drifted evidence and
// fails, never only for slices.
package translate

import "strings"

// RepresentationFamilies maps each proof-key family to its closed
// candidate set. Key families are recognized by classifyRepresentation.
var representationFamilies = map[string]map[string]bool{
	"slice-local": {
		"native-array":    true,
		"goslice-carrier": true,
	},
	"carrier": {
		"boolean": true,
		"js-string(equality-only-ordering-unsupported)":      true,
		"object-identity-nilable(undefined)":                 true,
		"class-instance-value(copy-on-bind,in-place-store)":  true,
		"js-closure-nilable(undefined)-capture-by-reference": true,
		"iface-box-nilable(undefined)-rtti-identity":         true,
		"js-map-nilable(undefined)-has-based-lookup":         true,
		"goslice-view-nilable(undefined)-conservative":       true,
		"number":                          true,
		"bigint-exact-64":                 true,
		"unit-literal-0":                  true,
		"native-array-of-element-carrier": true,
		"external-branded-handle":         true,
		"type-parameter-instantiation":    true,
	},
	"declaration": {
		"external-stub(typed-static, fail-closed)": true,
		"blank-var(no-initializer, no output)":     true,
		"ordered-effect(no binding)":               true,
		"const-folded-at-use":                      true,
		"class-direct-identity":                    true,
	},
}

// ClassifyRepresentation resolves one proof representation entry to its
// family and validates the candidate. It returns the family name and
// whether the candidate is a member of that family's closed set.
func ClassifyRepresentation(key, candidate string) (string, bool) {
	// Keys DECLARE their family by prefix; an unprefixed or unknown-
	// prefixed key fails, and each family admits only its own candidates.
	switch {
	case strings.HasPrefix(key, "slice-local:"):
		return "slice-local", representationFamilies["slice-local"][candidate]
	case strings.HasPrefix(key, "carrier:"):
		return classifyCarrier(key, candidate)
	case strings.HasPrefix(key, "decl:"):
		return classifyDeclaration(key, candidate)
	}
	return "", false
}

func classifyDeclaration(key, candidate string) (string, bool) {
	if representationFamilies["declaration"][candidate] {
		return "declaration", true
	}
	// Prefixed forms validate their EXACT inner grammar against the
	// CARRIER family only — a declaration disposition can never nest.
	if inner, ok := cutWrapped(candidate, "erased-to-carrier("); ok {
		if _, valid := classifyCarrier("carrier:inner", inner); valid {
			return "declaration", true
		}
		return "", false
	}
	if inner, ok := cutWrapped(candidate, "module-let(live-binding,"); ok {
		if _, valid := classifyCarrier("carrier:inner", inner); valid {
			return "declaration", true
		}
		return "", false
	}
	return "", false
}

func classifyCarrier(key, candidate string) (string, bool) {
	_ = key
	if bits, ok := strings.CutPrefix(candidate, "number-wrapped-"); ok {
		switch bits {
		case "8", "16", "32":
			return "carrier", true
		}
		return "", false
	}
	if representationFamilies["carrier"][candidate] {
		return "carrier", true
	}
	return "", false
}

// cutWrapped extracts the inner value of "prefix(inner)".
func cutWrapped(candidate, prefix string) (string, bool) {
	rest, ok := strings.CutPrefix(candidate, prefix)
	if !ok || !strings.HasSuffix(rest, ")") {
		return "", false
	}
	return strings.TrimSuffix(rest, ")"), true
}
