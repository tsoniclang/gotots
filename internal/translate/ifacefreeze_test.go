// Two-phase interface/external-universe freeze: an external implementer
// of an owned interface that is boxed only in a LATER body than a body
// which dispatches over that interface must still be a member of the
// interface's closed union — the union is resolved over the frozen
// whole-unit universe, independent of body order.
package translate_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

func TestOracleExternalImplementerDiscoveredLate(t *testing.T) {
	// dispatchFirst (which resolves the Stringy union) is declared BEFORE
	// boxLater (which first boxes *bytes.Buffer, an external Stringy
	// implementer). Without the freeze, dispatchFirst's union would omit
	// *bytes.Buffer and the late-boxed value would hit the unreachable
	// default; with it, dispatch is exact.
	result, err := oracle.RunAssembled(t.TempDir(), map[string]string{"fixture": `package fixture

import "bytes"

type Stringy interface {
	String() string
}

func dispatchFirst(s Stringy) string {
	return s.String()
}

func boxLater() Stringy {
	var b bytes.Buffer
	b.WriteString("late")
	return &b
}

func LateImplementer() string {
	return dispatchFirst(boxLater())
}
`}, map[string]string{
		"bytes": `import { type GoExtern } from "../../language-abi/goextern.js";
class BufferEmu { buf = ""; }
type Handle = GoExtern<"bytes.Buffer">;
export function Buffer$goZero$(): Handle { return new BufferEmu() as Handle; }
export function Buffer$goClone$(v: Handle): Handle { const b = v as unknown as BufferEmu; const o = new BufferEmu(); o.buf = b.buf; return o as Handle; }
export function Buffer$goSet$(dst: Handle, src: Handle): void { (dst as unknown as BufferEmu).buf = (src as unknown as BufferEmu).buf; }
export function Buffer$WriteString(recv: Handle | undefined, s: string): readonly [bigint, undefined] { const b = recv as unknown as BufferEmu; b.buf += s; return [BigInt(s.length), undefined]; }
export function Buffer$String(recv: Handle | undefined): string { return (recv as unknown as BufferEmu).buf; }
`,
	})
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("differential mismatch:\n--- go ---\n%s--- generated ---\n%s", result.GoOutput, result.TSOutput)
	}
}
