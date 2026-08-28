package callableimplementation

import "testing"

func TestStagedVerificationAllowsStaticallySafeDirectBroadCall(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		source string
	}{
		{
			name: "call",
			source: "export function addFast(value: number): number {\n" +
				"  return globalThis.Number(BigInt.asIntN(64, BigInt(value)));\n" +
				"}\n",
		},
		{
			name: "construction",
			source: "export function addFast(value: number): number {\n" +
				"  const boxed = new globalThis.Number(value);\n" +
				"  return boxed.valueOf();\n" +
				"}\n",
		},
		{
			name: "tagged template",
			source: "export function addFast(value: number): number {\n" +
				"  return globalThis.String.raw`${value}`.length;\n" +
				"}\n",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newStagedVerificationFixture(t)
			verified, err := VerifyStagedGeneratedContracts(fixture.config(
				t,
				testCase.source,
				[]string{"addFast"},
			))
			if err != nil {
				t.Fatal(err)
			}
			if len(verified) != 1 {
				t.Fatalf("verified implementations = %d, want 1", len(verified))
			}
		})
	}
}
