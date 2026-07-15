// Vertical-slice contract tests: Go semantic differential oracles for the
// reviewed v1 subset, fail-closed diagnostics for constructs outside it,
// and deterministic generation.
package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
	"github.com/tsoniclang/gotots/internal/translate"
)

func runOracle(t *testing.T, fixtureSource string) *oracle.Result {
	t.Helper()
	result, err := oracle.Run(t.TempDir(), map[string]string{"fixture": fixtureSource})
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("differential mismatch across %d cases:\n--- go ---\n%s--- generated ---\n%s",
			result.Cases, result.GoOutput, result.TSOutput)
	}
	return result
}

func TestOracleIntegerSemantics(t *testing.T) {
	runOracle(t, `package fixture

func AddInt32Overflow() int32 {
	var a int32 = 2147483647
	var b int32 = 1
	return a + b
}

func MulInt32Wraps() int32 {
	var a int32 = 123456789
	var b int32 = 371
	return a * b
}

func DivTruncatesTowardZero() int32 {
	var a int32 = -7
	var b int32 = 2
	return a / b
}

func RemFollowsDividend() int32 {
	var a int32 = -7
	var b int32 = 2
	return a % b
}

func MinInt32DivMinusOneWraps() int32 {
	var a int32 = -2147483648
	var b int32 = -1
	return a / b
}

func Int64Wraps() int64 {
	var a int64 = 9223372036854775807
	var b int64 = 1
	return a + b
}

func Uint8Wraps() uint8 {
	var a uint8 = 250
	var b uint8 = 10
	return a + b
}

func Uint64Wraps() uint64 {
	var a uint64 = 18446744073709551615
	var b uint64 = 2
	return a + b
}

func ShiftBeyondWidth() int32 {
	var a int32 = 1
	var n int = 40
	return a << n
}

func SignedShiftFills() int32 {
	var a int32 = -8
	var n int = 1
	return a >> n
}

func UnsignedShift() uint32 {
	var a uint32 = 2147483648
	var n int = 4
	return a >> n
}

func BitwiseOps() (int32, uint32, int64) {
	var a int32 = -12345
	var b int32 = 6789
	var c uint32 = 4000000000
	var d uint32 = 123456789
	var e int64 = -9999999999
	var f int64 = 1234567
	return a & b, c ^ d, e &^ f
}

func Conversions() (int32, uint8, int64, uint32) {
	var wide int64 = 300
	var negative int32 = -1
	return int32(wide), uint8(wide), int64(uint8(200)), uint32(negative)
}

func Complement() (int32, uint64) {
	var a int32 = 5
	var b uint64 = 5
	return ^a, ^b
}

func Negation() (int32, int64) {
	var a int32 = -2147483648
	var b int64 = 42
	return -a, -b
}
`)
}

func TestOraclePanics(t *testing.T) {
	runOracle(t, `package fixture

func DivideByZeroInt32() int32 {
	var a int32 = 1
	var b int32 = 0
	return a / b
}

func DivideByZeroInt64() int64 {
	var a int64 = 1
	var b int64 = 0
	return a / b
}

func RemByZero() int32 {
	var a int32 = 1
	var b int32 = 0
	return a % b
}

func NegativeShift() int32 {
	var a int32 = 1
	var n int = -1
	return a << n
}
`)
}

func TestOracleControlFlowAndAssignment(t *testing.T) {
	runOracle(t, `package fixture

func Swap() (int, int) {
	a, b := 1, 2
	a, b = b, a
	return a, b
}

func RotateThree() (int32, int32, int32) {
	var a int32 = 1
	var b int32 = 2
	var c int32 = 3
	a, b, c = c, a, b
	return a, b, c
}

func LoopSum() int {
	total := 0
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			total += i
		}
	}
	return total
}

func ContinueSkips() int {
	total := 0
	for i := 0; i < 5; i++ {
		if i == 2 {
			continue
		}
		total += i
	}
	return total
}

func BreakStops() int {
	total := 0
	for {
		total++
		if total == 7 {
			break
		}
	}
	return total
}

func IfInitScopes() int32 {
	var outer int32 = 1
	if inner := outer + 41; inner > 10 {
		return inner
	}
	return outer
}

func NestedElse() string {
	value := 15
	if value < 10 {
		return "small"
	} else if value < 20 {
		return "medium"
	} else {
		return "large"
	}
}

func StringBuild() string {
	text := "a"
	for i := 0; i < 3; i++ {
		text += "b"
	}
	return text
}

func BoolLogic() bool {
	isTrue := true
	isFalse := false
	return (isTrue || isFalse) && !isFalse
}

func StringEquality() (bool, bool) {
	left := "héllo"
	right := "héllo"
	return left == right, left != "other"
}
`)
}

func TestOracleCallsAndMultiResults(t *testing.T) {
	runOracle(t, `package fixture

func pair() (int32, string) {
	return 42, "answer"
}

func triple() (int32, string, bool) {
	return 7, "x", true
}

func UsePair() int32 {
	value, label := pair()
	if label == "answer" {
		return value
	}
	return 0
}

func ForwardTriple() (int32, string, bool) {
	return triple()
}

func ReassignFromCall() (int32, string) {
	value, label := pair()
	value, label = pair()
	return value + 1, label
}

func CallChain() int32 {
	return add(add(1, 2), add(3, 4))
}

func add(a int32, b int32) int32 {
	return a + b
}

func Float64Math() (float64, float64, bool) {
	a := 1.5
	b := 0.25
	quotient := a / 0.0
	return a*b + 0.125, quotient, quotient > 1000000.0
}
`)
}

func TestFailClosedDiagnostics(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		code    string
		mention string
	}{
		{
			name: "select statement",
			source: `package fixture
func Case() { select {} }
`,
			code: "GOTOTS_UNSUPPORTED_STATEMENT", mention: "SelectStmt",
		},
		{
			name: "goto",
			source: `package fixture
func Case() int {
	i := 0
loop:
	i++
	if i < 3 {
		goto loop
	}
	return i
}
`,
			code: "GOTOTS_UNSUPPORTED_STATEMENT", mention: "LabeledStmt",
		},
		{
			name: "struct value equality",
			source: `package fixture
type Point struct{ X int32 }
func Case() bool {
	a := Point{X: 1}
	b := Point{X: 1}
	return a == b
}
`,
			code: "GOTOTS_UNSUPPORTED_OPERATION", mention: "equality on",
		},
		{
			name: "full slice expression",
			source: `package fixture
func Case() int {
	values := []int{1, 2, 3}
	limited := values[0:1:2]
	return int(len(limited))
}
`,
			code: "GOTOTS_UNSUPPORTED_EXPRESSION", mention: "full slice expression",
		},
		{
			name: "external call with an unreviewed signature",
			source: `package fixture
import "os"
func Case() bool { f, err := os.Open("x"); return f != nil && err == nil }
`,
			code: "GOTOTS_UNSUPPORTED_EXPRESSION", mention: "call outside the translated unit",
		},
		{
			name: "defer below top level",
			source: `package fixture
func Case(enabled bool) int {
	if enabled {
		defer helper()
	}
	return 1
}
func helper() {}
`,
			code: "GOTOTS_UNSUPPORTED_STATEMENT", mention: "defer below the function's top-level block",
		},
		{
			name: "anonymous struct type",
			source: `package fixture
func Case() int32 {
	point := struct{ X int32 }{X: 1}
	return point.X
}
`,
			code: "GOTOTS_UNSUPPORTED_TYPE", mention: "struct type",
		},
		{
			name: "embedded field",
			source: `package fixture
type Base struct{ N int32 }
type Derived struct{ Base }
`,
			code: "GOTOTS_UNSUPPORTED_DECLARATION", mention: "embedded field",
		},
		{
			name: "float map key",
			source: `package fixture
func Case() int32 {
	m := make(map[float64]int32)
	m[1.5] = 1
	return m[1.5]
}
`,
			code: "GOTOTS_UNSUPPORTED_TYPE", mention: "map key type float64",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := oracle.Run(t.TempDir(), map[string]string{"fixture": c.source})
			if err == nil {
				t.Fatalf("expected fail-closed diagnostic %s", c.code)
			}
			if !strings.Contains(err.Error(), c.code) {
				t.Fatalf("expected diagnostic code %s, got: %v", c.code, err)
			}
			if !strings.Contains(err.Error(), c.mention) {
				t.Fatalf("expected diagnostic to mention %q, got: %v", c.mention, err)
			}
		})
	}
}

func translateFixture(t *testing.T, source string) *translate.Generated {
	t.Helper()
	generated, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": source})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	return generated
}

func TestGenerationIsDeterministic(t *testing.T) {
	source := `package fixture

func Value() (int32, string) {
	total := int32(0)
	for i := int32(0); i < 5; i++ {
		total += i
	}
	return total, "done"
}
`
	first := translateFixture(t, source)
	second := translateFixture(t, source)
	if len(first.Files) != len(second.Files) {
		t.Fatalf("file inventories differ: %d vs %d", len(first.Files), len(second.Files))
	}
	for path, content := range first.Files {
		if second.Files[path] != content {
			t.Errorf("nondeterministic generation for %s", path)
		}
	}
	if len(first.Proofs) != 1 || first.Proofs[0].LoweringPlan != translate.LoweringPlanV1 {
		t.Fatalf("unexpected proof records: %+v", first.Proofs)
	}
	proof := first.Proofs[0]
	if proof.BodyHash == "" || proof.SignatureHash == "" || len(proof.Operations) == 0 {
		t.Fatalf("proof record incomplete: %+v", proof)
	}
	if len(proof.Representations) == 0 {
		t.Fatalf("proof record missing representations: %+v", proof)
	}
	for path, owner := range first.Ownership {
		if owner != "generated-core" && owner != "generated-language-abi" {
			t.Errorf("unexpected ownership %q for %s", owner, path)
		}
	}
}
