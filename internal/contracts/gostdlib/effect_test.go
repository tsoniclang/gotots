package gostdlib_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestEffectKindIsClosedToSynchronous(t *testing.T) {
	tests := []struct {
		effect gostdlib.EffectKind
		valid  bool
	}{
		{effect: gostdlib.EffectInvalid, valid: false},
		{effect: gostdlib.EffectSynchronous, valid: true},
		{effect: gostdlib.EffectKind("async"), valid: false},
		{effect: gostdlib.EffectKind("awaitable"), valid: false},
	}
	for _, test := range tests {
		if got := test.effect.Valid(); got != test.valid {
			t.Fatalf("effect %q Valid() = %t, want %t", test.effect, got, test.valid)
		}
	}
}
