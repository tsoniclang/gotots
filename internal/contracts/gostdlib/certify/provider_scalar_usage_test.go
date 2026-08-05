package certify

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestCallableScalarAliasesRejectSameCarrierWrongIdentity(t *testing.T) {
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(types.NewVar(
			token.NoPos,
			nil,
			"value",
			types.Typ[types.Int],
		)),
		types.NewTuple(types.NewVar(
			token.NoPos,
			nil,
			"result",
			types.Typ[types.Int],
		)),
		false,
	)
	err := verifyCallableScalarAliases(
		"example.com/provider|F",
		signature,
		gostdlib.AccessExport,
		tsgo.ProjectCallableScalarAliases{
			Parameters: [][]string{{"int64"}},
			Results:    []string{"int64"},
		},
	)
	if err == nil || !strings.Contains(
		err.Error(),
		"parameter 0 scalar aliases are [int64], want [int]",
	) {
		t.Fatalf("scalar identity error = %v", err)
	}
}

func TestSourceScalarAliasesPreserveNestedGoIdentities(t *testing.T) {
	callback := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(types.NewVar(
			token.NoPos,
			nil,
			"byteValue",
			types.Typ[types.Uint8],
		)),
		types.NewTuple(types.NewVar(
			token.NoPos,
			nil,
			"nativeValue",
			types.Typ[types.Int],
		)),
		false,
	)
	source := types.NewTuple(
		types.NewVar(token.NoPos, nil, "wide", types.Typ[types.Int64]),
		types.NewVar(token.NoPos, nil, "callback", callback),
		types.NewVar(
			token.NoPos,
			nil,
			"pointer",
			types.NewPointer(types.Typ[types.Bool]),
		),
	)
	actual := sourceScalarAliases(source)
	expected := []string{"int64", "uint8", "int", "bool", "bool"}
	if !slicesEqual(actual, expected) {
		t.Fatalf("scalar aliases = %v, want %v", actual, expected)
	}
}

func TestProviderStructFieldRejectsSameCarrierWrongScalarIdentity(t *testing.T) {
	err := verifyProviderStructFieldScalarAliases(
		"Type.Size_",
		types.Typ[types.Uintptr],
		[]string{"uint64"},
	)
	if err == nil || !strings.Contains(
		err.Error(),
		"target scalar aliases are [uint64], want [uintptr]",
	) {
		t.Fatalf("field scalar identity error = %v", err)
	}
}

func slicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
