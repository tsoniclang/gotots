package emit_test

import (
	"strings"
	"testing"
)

func waveNineFunctionWithPrefix(
	t *testing.T,
	printed string,
	prefix string,
) string {
	t.Helper()
	start := strings.Index(printed, "function "+prefix)
	if start < 0 {
		t.Fatalf("generated output lacks function prefix %s:\n%s", prefix, printed)
	}
	start = strings.LastIndex(printed[:start], "export ")
	if start < 0 {
		t.Fatalf("generated function prefix %s is not exported", prefix)
	}
	rest := printed[start:]
	next := strings.Index(rest[len("export "):], "\nexport ")
	if next < 0 {
		return rest
	}
	return rest[:len("export ")+next]
}

func waveNineClassMemberText(
	t *testing.T,
	printed string,
	className string,
	marker string,
) string {
	t.Helper()
	classStart := strings.Index(printed, "export class "+className)
	if classStart < 0 {
		t.Fatalf("generated output lacks class %s", className)
	}
	memberOffset := strings.Index(printed[classStart:], marker)
	if memberOffset < 0 {
		t.Fatalf("generated class %s lacks member marker %q", className, marker)
	}
	memberStart := classStart + memberOffset + 1
	memberEnd := strings.Index(printed[memberStart:], "\n    }\n")
	if memberEnd < 0 {
		t.Fatalf("generated class %s member %q has no boundary", className, marker)
	}
	return printed[memberStart : memberStart+memberEnd+len("\n    }")]
}
