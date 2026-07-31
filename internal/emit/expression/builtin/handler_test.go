package builtin

import (
	"go/token"
	"go/types"
	"testing"
)

func TestFromObjectOwnsBuiltinIdentityClassification(t *testing.T) {
	length := types.Universe.Lookup("len")
	builtin, ok := FromObject(length)
	if !ok || types.Object(builtin) != length {
		t.Fatalf("builtin classification = %v, %t", builtin, ok)
	}
	variable := types.NewVar(
		token.NoPos,
		types.NewPackage("example.com/source", "source"),
		"len",
		types.Typ[types.Int],
	)
	if builtin, ok := FromObject(variable); ok || builtin != nil {
		t.Fatalf("ordinary object classified as builtin: %v", builtin)
	}
}
