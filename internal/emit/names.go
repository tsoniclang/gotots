package emit

// reservedTSNames are identifiers that are legal in Go but reserved or
// hazardous as bindings in strict-mode ECMAScript modules — including
// names that would shadow globals the generated code relies on
// (undefined, NaN, Infinity, globalThis). Go keywords never appear here
// because they can never be Go identifiers.
var reservedTSNames = map[string]bool{
	"arguments": true, "await": true, "catch": true, "class": true,
	"debugger": true, "delete": true, "do": true, "enum": true,
	"eval": true, "export": true, "extends": true, "false": true,
	"finally": true, "function": true, "globalThis": true, "implements": true,
	"in": true, "Infinity": true, "instanceof": true, "let": true,
	"NaN": true, "new": true, "null": true, "private": true,
	"protected": true, "public": true, "static": true, "super": true,
	"this": true, "throw": true, "true": true, "try": true,
	"typeof": true, "undefined": true, "void": true, "while": true,
	"with": true, "yield": true,
}

// tsName spells one source-named binding in generated TypeScript.
// Reserved or hazardous identifiers gain a "$" suffix; Go identifiers
// can never contain "$", so escaped names never collide with any source
// name, and declaration and reference sites share this single mapping.
func tsName(name string) string {
	if reservedTSNames[name] {
		return name + "$"
	}
	return name
}
