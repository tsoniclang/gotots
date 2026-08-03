package gostdlib_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestEffectMaySuspend(t *testing.T) {
	tests := []struct {
		effect gostdlib.EffectKind
		want   bool
	}{
		{effect: gostdlib.EffectInvalid, want: false},
		{effect: gostdlib.EffectSynchronous, want: false},
		{effect: gostdlib.EffectAsynchronous, want: true},
		{effect: gostdlib.EffectAwaitable, want: true},
	}
	for _, test := range tests {
		if got := test.effect.MaySuspend(); got != test.want {
			t.Fatalf("effect %q MaySuspend() = %t, want %t", test.effect, got, test.want)
		}
	}
}
