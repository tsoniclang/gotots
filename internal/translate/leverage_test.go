// Contract tests for the package-completion batch: labeled branches,
// break inside type switches, panic with error values, struct equality
// through the canonical key, interface-to-interface assertions, and
// external variable reads.
package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

func TestOracleLabeledBranches(t *testing.T) {
	runOracle(t, `package fixture

func LabeledBreak() int {
	total := 0
outer:
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if i*j >= 6 {
				break outer
			}
			total += i * 10 + j
		}
	}
	return total
}

func LabeledContinue() int {
	total := 0
next:
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if j > i {
				continue next
			}
			total++
		}
	}
	return total
}

func LabeledRangeBreak() int {
	values := []int{1, 2, 3, 4}
scan:
	for _, v := range values {
		switch v {
		case 3:
			break scan
		}
	}
	return len(values)
}

type shape interface {
	area() int
}

type square struct {
	side int
}

func (s square) area() int { return s.side * s.side }

func BreakInTypeSwitch() int {
	var v any = square{side: 3}
	total := 0
	switch v.(type) {
	case square:
		total = 1
		if total == 1 {
			break
		}
		total = 2
	}
	return total
}
`)
}

func TestOraclePanicWithError(t *testing.T) {
	runOracle(t, `package fixture

type parseError struct {
	detail string
}

func (e *parseError) Error() string {
	return "parse failed: " + e.detail
}

func PanicWithError() int {
	var err error = &parseError{detail: "bad token"}
	panic(err)
}

func PanicWithNilError() int {
	var err error
	panic(err)
}
`)
}

func TestOracleStructEqualityAndIfaceAssert(t *testing.T) {
	runOracle(t, `package fixture

type point struct {
	x int
	y int
	label string
}

func StructEquality() (bool, bool, bool) {
	a := point{x: 1, y: 2, label: "p"}
	b := point{x: 1, y: 2, label: "p"}
	c := point{x: 1, y: 3, label: "p"}
	return a == b, a == c, b != c
}

type reader interface {
	read() string
}

type closer interface {
	close() int
}

type file struct {
	name string
}

func (f *file) read() string { return "data:" + f.name }
func (f *file) close() int   { return 1 }

func IfaceToIfaceAssert() (string, int, bool) {
	var r reader = &file{name: "a.txt"}
	c, ok := r.(closer)
	got := 0
	if ok {
		got = c.close()
	}
	both := r.(closer)
	return r.read(), got + both.close(), ok
}

type writer interface {
	write() int
}

func IfaceAssertMissingMethodPanics() int {
	var r reader = &file{name: "x"}
	w := r.(writer)
	return w.write()
}
`)
}

func TestExternalVarReads(t *testing.T) {
	generated, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": `package fixture

import "io/fs"

func Case(err error) bool {
	return err == fs.SkipDir
}
`})
	if err != nil {
		t.Fatalf("translation must succeed statically: %v", err)
	}
	var core string
	for path, content := range generated.Files {
		if strings.Contains(path, "fixture/package.ts") {
			core = content
		}
	}
	if !strings.Contains(core, `goext$.goExternalCall("io/fs.SkipDir", [])`) {
		t.Errorf("expected external variable contract dispatch, got:\n%s", core)
	}
}

func TestOracleEmbeddedFields(t *testing.T) {
	runOracle(t, `package fixture

type base struct {
	id   int
	name string
}

func (b *base) describe() string {
	return b.name
}

func (b base) ident() int {
	return b.id
}

type derived struct {
	base
	extra int
}

func PromotedFieldAccess() (int, string, int) {
	d := derived{base: base{id: 7, name: "d"}, extra: 3}
	return d.id, d.name, d.extra
}

func PromotedFieldStore() int {
	d := derived{}
	d.id = 41
	d.base.id += 1
	return d.id
}

func PromotedMethodCalls() (string, int) {
	d := derived{base: base{id: 9, name: "emb"}}
	return d.describe(), d.ident()
}

func PromotedPointerReceiverMutation() string {
	d := &derived{}
	d.name = "before"
	d.describe()
	d.base.name = "after"
	return d.describe()
}

type deep struct {
	derived
}

func TwoLevelPromotion() (int, string) {
	x := deep{derived: derived{base: base{id: 5, name: "deep"}}}
	return x.id, x.describe()
}

func EmbeddedValueCopySemantics() (int, int) {
	d := derived{base: base{id: 1}}
	e := d
	e.id = 99
	return d.id, e.id
}
`)
}

func TestOraclePromotedInterfaceDispatch(t *testing.T) {
	runOracle(t, `package fixture

type namer interface {
	name() string
}

type core struct {
	label string
}

func (c *core) name() string {
	return "core:" + c.label
}

type wrapper struct {
	core
	extra int
}

func PromotedDispatchThroughInterface() (string, string) {
	w := &wrapper{core: core{label: "x"}}
	var n namer = w
	direct := w.name()
	return n.name(), direct
}

func PromotedMethodSetAssertion() (bool, string) {
	var v any = &wrapper{core: core{label: "y"}}
	n, ok := v.(namer)
	if !ok {
		return false, ""
	}
	return ok, n.name()
}
`)
}
