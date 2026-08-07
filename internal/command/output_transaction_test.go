package command

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOutputTransactionPreservesPreviousBuildOnWriteFailure(t *testing.T) {
	target := filepath.Join(t.TempDir(), "generated")
	writeCommandFixture(t, filepath.Join(target, "previous.ts"), "export const previous = true;\n")
	writeFailure := errors.New("injected output failure")
	_, err := writeOutputTransaction(target, func(staging string) (int, error) {
		writeCommandFixture(t, filepath.Join(staging, "partial.ts"), "export const partial = true;\n")
		return 0, writeFailure
	})
	if !errors.Is(err, writeFailure) {
		t.Fatalf("write failure = %v, want injected failure", err)
	}
	if _, err := os.Stat(filepath.Join(target, "previous.ts")); err != nil {
		t.Fatalf("previous output was not preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "partial.ts")); !os.IsNotExist(err) {
		t.Fatalf("partial output became observable: %v", err)
	}
}
