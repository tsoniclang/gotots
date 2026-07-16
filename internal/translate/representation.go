// The versioned representation registry: the closed vocabulary of every
// candidate a proof may record, per semantic family. Proof records join
// to this registry at the acceptance gate — an unknown key family or an
// unknown candidate in ANY family is forged or drifted evidence and
// fails, never only for slices.
package translate

import "strings"

// RepresentationFamilies maps each proof-key family to its closed
// candidate set. Key families are recognized by classifyRepresentation.
var RepresentationFamilies = map[string]map[string]bool{
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
	"declaration-prefixed": {
		"erased-to-carrier": true,
		"module-let":        true,
		"number-wrapped":    true,
	},
}

// ClassifyRepresentation resolves one proof representation entry to its
// family and validates the candidate. It returns the family name and
// whether the candidate is a member of that family's closed set.
func ClassifyRepresentation(key, candidate string) (string, bool) {
	if strings.HasPrefix(key, "slice-local:") {
		return "slice-local", RepresentationFamilies["slice-local"][candidate]
	}
	if RepresentationFamilies["declaration"][candidate] {
		return "declaration", true
	}
	for prefix := range RepresentationFamilies["declaration-prefixed"] {
		if strings.HasPrefix(candidate, prefix+"(") {
			return "declaration-prefixed", true
		}
	}
	if RepresentationFamilies["carrier"][candidate] {
		return "carrier", true
	}
	for prefix := range RepresentationFamilies["declaration-prefixed"] {
		if strings.HasPrefix(candidate, prefix+"-") {
			return "declaration-prefixed", true
		}
	}
	return "", false
}
