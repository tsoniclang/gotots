package emit

import "github.com/tsoniclang/gotots/internal/tsident"

// tsName spells one source-named identifier in generated TypeScript
// through the single authoritative identifier policy: reserved or
// hazardous identifiers gain a "$" suffix; Go identifiers can never
// contain "$", so escaped names never collide with any source name, and
// declaration and reference sites share this one mapping.
func tsName(name string) string { return tsident.Escape(name) }
