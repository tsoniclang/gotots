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
