// Addressability contract tests from review round five: taking the
// address of a range, type-switch, or named-result binding must never
// emit an undeclared cell — it is either boxed correctly (ordinary
// locals) or fails closed.
package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

func TestOrdinaryLocalBoxingStillWorks(t *testing.T) {
	// A plain local whose address is taken boxes exactly.
	runOracle(t, `package fixture

func LocalAddress() int {
	x := 3
	p := &x
	*p = *p + 4
	return x
}
`)
}

func TestBoxedBindingsFailClosed(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		mention string
	}{
		{
			name: "address of an integer range variable",
			source: `package fixture
func Case() int {
	total := 0
	for i := range 3 {
		p := &i
		total += *p
	}
	return total
}
`,
			mention: "address of a range variable",
		},
		{
			name: "address of a slice range value",
			source: `package fixture
func Case() int {
	sum := 0
	for _, v := range []int{1, 2, 3} {
		p := &v
		sum += *p
	}
	return sum
}
`,
			mention: "address of a range variable",
		},
		{
			name: "address of a type-switch binding",
			source: `package fixture
func Case() int {
	var v any = 5
	switch n := v.(type) {
	case int:
		p := &n
		return *p
	}
	return 0
}
`,
			mention: "address of a type-switch variable",
		},
		{
			name: "address of a named result",
			source: `package fixture
func Case() (n int) {
	p := &n
	*p = 7
	return
}
`,
			mention: "address of a named result",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": c.source})
			if err == nil {
				t.Fatalf("expected a fail-closed diagnostic mentioning %q", c.mention)
			}
			if !strings.Contains(err.Error(), c.mention) {
				t.Fatalf("expected diagnostic to mention %q, got: %v", c.mention, err)
			}
		})
	}
}
