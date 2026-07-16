// Package-initialization edge tests from review round five: a package
// imported only for folded constants or type-only uses must still be
// imported (evaluated), so its initialization side effects run.
package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

func TestInitEdgeSurvivesConstantFolding(t *testing.T) {
	generated, _, err := oracle.Translate(t.TempDir(), map[string]string{
		"fixture": `package fixture

import "oracle.fixture/dep"

func Case() int { return dep.C }
`,
		"dep": `package dep

const C = 41

var Effect = compute()

func compute() int { return 7 }
`,
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	core := generated.Files["core/oracle.fixture/fixture/package.ts"]
	if !strings.Contains(core, `from "../dep/package.js"`) {
		t.Errorf("consumer dropped the init edge to dep despite folding dep.C:\n%s", core)
	}
}
