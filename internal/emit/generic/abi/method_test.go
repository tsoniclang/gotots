package abi

import (
	"slices"
	"testing"
)

func TestMethodABIExactJoinRejectsReceiverCapabilitySwap(t *testing.T) {
	declaration := Method(
		[]string{"capability:equal"},
		"receiver",
		[]string{"source:0"},
	)
	invocation := Method(
		[]string{"capability:equal"},
		"receiver",
		[]string{"source:0"},
	)
	if !slices.Equal(declaration, invocation) {
		t.Fatalf(
			"method ABI identities do not join: declaration=%v invocation=%v",
			declaration,
			invocation,
		)
	}
	receiverFirstMutation := []string{
		"receiver",
		"capability:equal",
		"source:0",
	}
	if slices.Equal(declaration, receiverFirstMutation) {
		t.Fatal("receiver-first method ABI mutation was not distinguished")
	}
}
