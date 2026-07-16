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
