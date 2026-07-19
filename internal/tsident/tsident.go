// Package tsident is the single authoritative TypeScript-identifier policy
// shared by IR binding-name allocation and emission. It owns two concerns:
// reserved-word escaping (a source identifier that is a reserved or
// hazardous ECMAScript name gains a "$" suffix), and the registry of host
// globals the emitter injects into generated per-package code. Because a
// legal Go binding may share a host global's spelling, every injected
// reference MUST be qualified through globalThis so it is intrinsically
// unshadowable; the registry exists so a structural gate can prove no bare
// injection remains.
package tsident

import "sort"

// reserved identifiers are legal in Go but reserved or hazardous as
// bindings in strict-mode ECMAScript modules (globals the generated
// runtime relies on included). Go identifiers can never contain "$", so an
// escaped name never collides with a source name, and declaration and
// reference sites share this single mapping.
var reserved = map[string]bool{
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

// Escape returns the emission spelling of one source-named binding.
func Escape(name string) string {
	if reserved[name] {
		return name + "$"
	}
	return name
}

// Reserved reports whether a source spelling is escaped.
func Reserved(name string) bool { return reserved[name] }

// hostGlobals are the JavaScript host globals whose bare identifier the
// generated per-package code must never inject: a source binding may
// legally carry the same spelling and would capture the reference. Every
// injection qualifies them through Global(...) instead. The set is the
// authority the structural gate checks against.
var hostGlobals = map[string]bool{
	"Number": true, "String": true, "Boolean": true, "BigInt": true,
	"Array": true, "Object": true, "Symbol": true, "Math": true,
	"JSON": true, "Map": true, "Set": true, "WeakMap": true,
	"WeakSet": true, "Error": true, "Promise": true, "RegExp": true,
	"Date": true, "parseInt": true, "parseFloat": true, "isNaN": true,
	"isFinite": true, "Reflect": true, "Proxy": true, "globalThis": true,
}

// IsHostGlobal reports whether a spelling is an injected host global.
func IsHostGlobal(name string) bool { return hostGlobals[name] }

// HostGlobals lists the injected host globals in deterministic order.
func HostGlobals() []string {
	names := make([]string, 0, len(hostGlobals))
	for name := range hostGlobals {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Global spells an intrinsically unshadowable reference to a host global:
// globalThis is reserved (a source binding of that spelling escapes to
// globalThis$), so globalThis.<name> can never resolve to source-owned
// code.
func Global(name string) string { return "globalThis." + name }
