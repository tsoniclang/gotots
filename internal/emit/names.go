package emit

import "github.com/tsoniclang/gotots/internal/tsident"

// tsName spells one identifier in generated TypeScript through the single
// authoritative DECLARATION policy: module-scope declarations and their
// references escape reserved spellings (except the decl-safe NaN/Infinity,
// which generated code never spells bare), and declaration and reference
// sites — core modules and external stubs alike — share this one mapping.
// LOCAL bindings never pass through here raw: the IR binding allocator
// baked their unique fully-escaped names, under which tsName is the
// identity.
func tsName(name string) string { return tsident.EscapeDeclared(name) }
