package certify

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestCallableParameterEvidenceAndProfileCoverageAreExact(t *testing.T) {
	callback := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int])),
		types.NewTuple(types.NewVar(token.NoPos, nil, "ok", types.Typ[types.Bool])),
		false,
	)
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "count", types.Typ[types.Int]),
			types.NewVar(token.NoPos, nil, "predicate", callback),
		),
		types.NewTuple(),
		false,
	)
	direct := []gostdlib.ProviderCallableParameterDocument{{
		Parameter: 1,
		Effect:    gostdlib.EffectSynchronous,
	}}
	if mismatches := callableParameterBindingMismatches(
		"example.Search",
		signature,
		direct,
	); len(mismatches) != 0 {
		t.Fatalf("direct provider mismatches = %v", mismatches)
	}
	asynchronous := []gostdlib.ProviderCallableParameterDocument{{
		Parameter: 1,
		Effect:    gostdlib.EffectAsynchronous,
	}}
	mismatches := callableParameterBindingMismatches(
		"example.Search",
		signature,
		asynchronous,
	)
	if len(mismatches) != 1 || !strings.Contains(mismatches[0], "want sync or awaitable") {
		t.Fatalf("asynchronous provider mismatches = %v", mismatches)
	}
	if mismatches := callableParameterBindingMismatches(
		"example.Search",
		signature,
		nil,
	); len(mismatches) != 1 || !strings.Contains(mismatches[0], "exact-join") {
		t.Fatalf("omitted provider evidence mismatches = %v", mismatches)
	}
	nested := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "callback", callback)),
		types.NewTuple(),
		false,
	)
	nestedOwner := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "outer", nested)),
		types.NewTuple(),
		false,
	)
	nestedMismatches := callableParameterBindingMismatches(
		"example.Nested",
		nestedOwner,
		[]gostdlib.ProviderCallableParameterDocument{{
			Parameter: 0,
			Effect:    gostdlib.EffectSynchronous,
		}},
	)
	if len(nestedMismatches) != 1 ||
		!strings.Contains(nestedMismatches[0], "nested callable transport is unsupported") {
		t.Fatalf("nested callable mismatches = %v", nestedMismatches)
	}

}
