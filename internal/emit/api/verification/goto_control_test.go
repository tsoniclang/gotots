package api_test

import (
	"go/token"
	"go/types"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestGotoTargetsUseExactLabelIdentity(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/labels", "labels")
	first := types.NewLabel(token.Pos(10), sourcePackage, "again")
	second := types.NewLabel(token.Pos(20), sourcePackage, "again")
	target, err := NewDirectGotoTarget(GotoTargetContinue, "$again")
	if err != nil {
		t.Fatal(err)
	}
	context := (Context{}).WithGotoTarget(first, target)
	selected, ok := context.GotoTarget(first)
	if !ok || selected.Label() != "$again" {
		t.Fatalf("exact label target = %#v, %t", selected, ok)
	}
	if _, ok := context.GotoTarget(second); ok {
		t.Fatal("same-spelling foreign label acquired the goto target")
	}
}

func TestGotoTargetRejectsInvalidStateAndCapability(t *testing.T) {
	if _, err := NewDirectGotoTarget(GotoTargetState, "$label"); err == nil {
		t.Fatal("state target was admitted as direct")
	}
	if _, err := NewStateGotoTarget("", "$state", 1); err == nil {
		t.Fatal("state target without dispatch identity was admitted")
	}
	if _, err := NewStateGotoTarget("$dispatch", "$state", -1); err == nil {
		t.Fatal("negative state target was admitted")
	}
}
