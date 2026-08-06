package gostdlib

import (
	"strconv"
	"strings"
)

// ImplementationDisposition is the mechanically certified behavior status of
// one behavior-bearing provider binding or facet, derived from the strict
// checked provider project. A certified binding is linkage evidence only;
// the disposition is the behavior evidence.
type ImplementationDisposition string

const (
	DispositionInvalid ImplementationDisposition = ""
	// DispositionImplemented certifies that exact behavior is supplied.
	DispositionImplemented ImplementationDisposition = "implemented"
	// DispositionProfileBoundary certifies that one named selected profile
	// intentionally excludes the behavior.
	DispositionProfileBoundary ImplementationDisposition = "profile-boundary"
	// DispositionPlaceholder certifies a typed throwing boundary that is
	// never publishable when used.
	DispositionPlaceholder ImplementationDisposition = "placeholder"
)

func (d ImplementationDisposition) Valid() bool {
	return d == DispositionImplemented ||
		d == DispositionProfileBoundary ||
		d == DispositionPlaceholder
}

// CanonicalPlaceholderModulePath is the provider-relative source path of the
// one canonical placeholder owner. Certification classifies a body as a
// placeholder only when its checked call evidence reaches this owner's
// canonical symbol; message strings never select behavior.
const CanonicalPlaceholderModulePath = "src/internal/runtime/placeholder.ts"

// CanonicalPlaceholderExport is the sole canonical placeholder symbol name.
const CanonicalPlaceholderExport = "providerPlaceholder"

// CanonicalPlaceholderDependency reports whether one certified
// implementation-site identity is a canonical placeholder symbol. The
// canonical owners are the sole placeholder constructors; identity is
// checked by declaring path and symbol name, never by message content.
func CanonicalPlaceholderDependency(identity string) bool {
	path, remainder, found := strings.Cut(identity, "#")
	if !found || path != CanonicalPlaceholderModulePath {
		return false
	}
	symbol, _, _ := strings.Cut(remainder, "@")
	return symbol == CanonicalPlaceholderExport ||
		symbol == CanonicalPlaceholderExport+"Error" ||
		symbol == CanonicalPlaceholderExport+"Message"
}

// ImplementationSiteIdentity is the canonical serialized identity of one
// checked implementation site: the provider-relative declaring source path,
// the checked symbol name, and the declaration node ordinal that
// disambiguates same-named members of distinct owners in one file.
func ImplementationSiteIdentity(
	sourcePath string,
	symbol string,
	declarationIndex uint32,
) string {
	return sourcePath + "#" + symbol + "@" +
		strconv.FormatUint(uint64(declarationIndex), 10)
}

// ImplementationDocument is the certified behavior evidence of one private
// provider implementation site: its closed disposition and conservative
// value-level dependency identities.
type ImplementationDocument struct {
	Identity     string                    `json:"identity"`
	Disposition  ImplementationDisposition `json:"disposition"`
	Dependencies []string                  `json:"dependencies,omitempty"`
}
