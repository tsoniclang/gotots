package staticness

import "testing"

func TestSweepFlagsErasedDispatch(t *testing.T) {
	files := map[string]string{
		"a.ts": "export function f(i: GoIface) { return goif$.goIfaceCall(i, \"M|abc\", []); }",
		"b.ts": "const table: Record<string, Function> = {};",
		"c.ts": "const fn = box.r.m[key] as Function;",
	}
	v := Sweep(files)
	counts := Counts(v)
	if counts["iface-name-dispatch"] != 1 {
		t.Errorf("iface-name-dispatch = %d; want 1", counts["iface-name-dispatch"])
	}
	if counts["record-function-table"] != 1 {
		t.Errorf("record-function-table = %d; want 1", counts["record-function-table"])
	}
	if counts["method-table-index"] != 1 {
		t.Errorf("method-table-index = %d; want 1", counts["method-table-index"])
	}
}

func TestSweepPassesDirectCalls(t *testing.T) {
	files := map[string]string{
		"a.ts": `export function f(x: Counter): number { return Counter$get(x); }`,
		"b.ts": `switch (box.r) { case A$rtti: return A$M(box.v as A); }`,
	}
	if v := Sweep(files); len(v) != 0 {
		t.Errorf("direct calls flagged: %+v", v)
	}
}

func TestSweepIgnoresComments(t *testing.T) {
	files := map[string]string{"a.ts": "// goIfaceCall is the old erased form"}
	if v := Sweep(files); len(v) != 0 {
		t.Errorf("comment flagged: %+v", v)
	}
}

// TestMutationInjectedDispatchIsCaught proves the sweep fails closed on
// injected erased dispatch — the mutation test the acceptance spec
// requires: forging a registry or erased call into generated output must
// be rejected, and no file-local form evades detection.
func TestMutationInjectedDispatchIsCaught(t *testing.T) {
	mutations := []struct {
		name    string
		line    string
		pattern string
	}{
		{"registry-record", "const table: Record<string, Function> = {};", "record-function-table"},
		{"registry-map", "const t = new Map<string, Function>();", "map-function-table"},
		{"name-dispatch", "return goif$.goIfaceCall(i, \"M|abc\", []);", "iface-name-dispatch"},
		{"method-table", "const fn = box.r.m[method];", "method-table-index"},
		{"func-invoke", "return goFuncInvoke(fn, args);", "erased-func-invoke"},
		{"external-registry", "return goExternalCall(\"strings.Clone\", [v]);", "external-registry-call"},
		{"computed-call", "return obj[methodName](arg);", "computed-member-call"},
		{"apply-dispatch", "return obj[key].apply(obj, args);", "computed-member-call"},
		{"eval", "return eval(source);", "eval"},
		{"reflect", "return Reflect.get(o, k);", "reflect-construct"},
		{"erased-func-type", "let f: Function = g;", "erased-function-type"},
	}
	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			v := Sweep(map[string]string{"core/pkg/package.ts": m.line})
			found := false
			for _, viol := range v {
				if viol.Pattern == m.pattern {
					found = true
				}
			}
			if !found {
				t.Errorf("injected %q not caught as %q; got %+v", m.line, m.pattern, v)
			}
		})
	}
}

// TestNoFileLocalSuppression proves an in-file allowlist comment cannot
// exempt a prohibited site.
func TestNoFileLocalSuppression(t *testing.T) {
	files := map[string]string{
		"a.ts": "// staticness-allow: registry\nconst t: Record<string, Function> = {};",
	}
	if len(Sweep(files)) == 0 {
		t.Error("a file-local allowlist comment must not suppress a prohibited site")
	}
}
