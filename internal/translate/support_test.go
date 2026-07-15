package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

// TestExhaustiveUnsupportedAccounting: one body with several distinct
// unsupported operations records every site — finding one never hides
// the others — and the package's runnable output is withheld.
func TestExhaustiveUnsupportedAccounting(t *testing.T) {
	_, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": `package fixture

func Case() int {
	ch := make(chan int)
	_ = ch
	select {}
	total := 0
	var arr [4]int
	total = total + arr[0]
	goto done
done:
	return total
}
`})
	if err == nil {
		t.Fatal("expected the fixture package to be withheld")
	}
	text := err.Error()
	if !strings.Contains(text, "withheld") {
		t.Fatalf("expected a withheld diagnostic, got: %v", err)
	}
	for _, mention := range []string{
		"channel type",   // make(chan int)
		"ast.SelectStmt", // select {}
		"array type",     // var arr [4]int
		"branch goto",    // goto done
	} {
		if !strings.Contains(text, mention) {
			t.Errorf("site %q missing from the exhaustive account:\n%s", mention, text)
		}
	}
}

// TestUnimplementedBodyDoesNotHideSiblings: an unimplemented body in one
// package never blocks analysis or translation of independent bodies —
// the ledger carries both states.
func TestUnimplementedBodyDoesNotHideSiblings(t *testing.T) {
	_, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": `package fixture

func Supported() int {
	return 41 + 1
}

func Unsupported() int {
	select {}
}
`})
	if err == nil {
		t.Fatal("expected the fixture package to be withheld")
	}
	if !strings.Contains(err.Error(), "ast.SelectStmt") {
		t.Fatalf("expected the select site on record, got: %v", err)
	}
}
