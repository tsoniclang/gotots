package defined_test

import (
	"strings"
	"testing"
)

func TestMethodBearingFixedWidthNumericUsesOneNativeRepresentation(
	t *testing.T,
) {
	target := compileDefinedSource(t, `package spelling

type Flags uint32

func (flags Flags) Has(mask Flags) bool { return flags&mask != 0 }
func (flags *Flags) Add(mask Flags) { *flags |= mask }

func Direct(flags, mask Flags) bool { return flags.Has(mask) }
func Expression() func(Flags, Flags) bool { return Flags.Has }
func Value(flags Flags) func(Flags) bool { return flags.Has }
func PointerExpression() func(*Flags, Flags) { return (*Flags).Add }
`)
	for _, required := range []string{
		"export enum Flags",
		"export function Flags_Has(flags: Flags, mask: Flags): bool",
		"export function Flags_Add(flags: Pointer<Flags> | undefined, mask: Flags): void",
		"return Flags_Has(flags, mask);",
		"return Flags_Has;",
		"return Flags_Add;",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("native method-bearing numeric lacks %q:\n%s", required, target)
		}
	}
	for _, forbidden := range []string{
		"export class Flags",
		"new Flags(",
		".$value",
		"flags.Has(",
	} {
		if strings.Contains(target, forbidden) {
			t.Fatalf("native method-bearing numeric retains %q:\n%s", forbidden, target)
		}
	}
}
