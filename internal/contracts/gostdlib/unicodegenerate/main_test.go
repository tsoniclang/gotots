package main

import (
	"strings"
	"testing"
	"unicode"
)

func TestRenderPreservesRangeTableShape(t *testing.T) {
	payload := string(render(nil, []table{{
		name: "sample",
		value: &unicode.RangeTable{
			R16:         []unicode.Range16{{Lo: 1, Hi: 9, Stride: 2}},
			R32:         []unicode.Range32{{Lo: 0x10000, Hi: 0x10008, Stride: 4}},
			LatinOffset: 1,
		},
	}}))
	for _, exact := range []string{
		"[0x000001, 0x000009, 2]",
		"[0x010000, 0x010008, 4]",
		"export const sampleLatinOffset = 1;",
	} {
		if !strings.Contains(payload, exact) {
			t.Fatalf("generated table missing %q:\n%s", exact, payload)
		}
	}
	if !strings.HasSuffix(payload, ";\n") || strings.HasSuffix(payload, "\n\n") {
		t.Fatalf("generated table must end in exactly one newline:\n%q", payload)
	}
}
