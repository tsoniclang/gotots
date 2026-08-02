package emit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
)

func TestPackageExportProjectionWaitsForLateGenericRequirements(
	t *testing.T,
) {
	directory := t.TempDir()
	for name, content := range map[string]string{
		"go.mod": "module example.com/exportquiescence\n\ngo 1.26.4\n",
		"source.go": `package exportquiescence

func Existing() int32 { return 1 }

func Transform[T any](value T) T {
	var result T
	result = value
	return result
}

func use() int32 { return Transform(int32(2)) }
`,
	} {
		if err := os.WriteFile(
			filepath.Join(directory, name),
			[]byte(content),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	if err := session.require(scope.Lookup("Existing")); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	if err := session.require(scope.Lookup("use")); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)

	builder := session.packageBuilders[program.Roots()[0]]
	if builder == nil || !builder.exportPublished {
		t.Fatal("package export projection was not published")
	}
	if session.artifacts.HasPending() ||
		session.requirements.hasPending() ||
		session.packageExports.hasPending() {
		t.Fatal("late generic requirements did not reach quiescence")
	}
	transform := scope.Lookup("Transform")
	if len(session.requirements.appliedFor(sourceArtifactOwner(transform))) == 0 {
		t.Fatal("late generic declaration acquired no requirements")
	}
}
