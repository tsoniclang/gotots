package environmentobligation_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestEnvironmentObligationsAreCanonicalAndImmutable(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/obligations\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package obligations

import (
	"fmt"
	"os"
	"slices"
	"unicode/utf8"
)

func Render() string {
	return fmt.Sprint(os.Args[0])
}

func RuneBoundary() byte {
	return utf8.RuneSelf
}

func Values(values []int32, input <-chan int32) int32 {
	var total int32
	for value := range slices.Values(values) {
		total += value + <-input
	}
	return total
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	roots := make([]emit.Root, 0, 3)
	for _, name := range []string{"Render", "RuneBoundary", "Values"} {
		root, rootErr := emit.NewRoot(scope.Lookup(name))
		if rootErr != nil {
			t.Fatal(rootErr)
		}
		roots = append(roots, root)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	emission, err := emit.CompileWithOptions(
		program,
		roots,
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	obligations := emission.EnvironmentObligations()
	if len(obligations) == 0 {
		t.Fatal("environment obligations are empty")
	}
	if len(emission.EnvironmentObligations()) != len(obligations) {
		t.Fatal("environment obligation accessor is not stable")
	}
	identities := make(map[string]struct{}, len(obligations))
	var sprint, args, runeSelf, values *emit.EnvironmentObligation
	for index := range obligations {
		obligation := &obligations[index]
		key := obligation.Identity() + "|" +
			obligation.TargetName() + "|" +
			obligation.TargetFingerprint()
		if _, duplicate := identities[key]; duplicate {
			t.Fatalf("duplicate environment obligation %q", key)
		}
		identities[key] = struct{}{}
		if obligation.PackageKind() != load.PackageStandardLibraryContract {
			t.Fatalf(
				"obligation %q package kind = %d",
				obligation.Identity(),
				obligation.PackageKind(),
			)
		}
		if len(obligation.ContractKey()) != 64 ||
			len(obligation.TargetFingerprint()) != 64 {
			t.Fatalf(
				"obligation %q keys = %q / %q",
				obligation.Identity(),
				obligation.ContractKey(),
				obligation.TargetFingerprint(),
			)
		}
		if strings.Contains(obligation.SourceLocation(), project) {
			t.Fatalf(
				"obligation %q retained checkout path %q",
				obligation.Identity(),
				obligation.SourceLocation(),
			)
		}
		if index != 0 &&
			environmentObligationOrder(obligations[index-1], *obligation) > 0 {
			t.Fatalf(
				"environment obligations are not canonical at %d",
				index,
			)
		}
		switch {
		case obligation.PackagePath() == "fmt" &&
			obligation.Name() == "Sprint":
			sprint = obligation
		case obligation.PackagePath() == "os" &&
			obligation.Name() == "Args":
			args = obligation
		case obligation.PackagePath() == "unicode/utf8" &&
			obligation.Name() == "RuneSelf":
			runeSelf = obligation
		case obligation.PackagePath() == "slices" &&
			obligation.Name() == "Values":
			values = obligation
		}
	}
	for name, obligation := range map[string]*emit.EnvironmentObligation{
		"fmt.Sprint":    sprint,
		"os.Args":       args,
		"utf8.RuneSelf": runeSelf,
		"slices.Values": values,
	} {
		if obligation == nil {
			t.Fatalf("environment obligation %s is absent", name)
		}
		if obligation.SourceSignature() == "" {
			t.Fatalf("environment obligation %s lacks source signature", name)
		}
	}
	if args.Kind() != emit.EnvironmentObligationState ||
		runeSelf.Kind() != emit.EnvironmentObligationConstantProjection {
		t.Fatalf(
			"obligation kinds args=%d runeSelf=%d",
			args.Kind(),
			runeSelf.Kind(),
		)
	}
	requirements := values.Requirements()
	if len(requirements) == 0 {
		t.Fatal("generic slices.Values obligation lacks typed requirements")
	}
	requirements[0] = "mutated"
	if values.Requirements()[0] == "mutated" {
		t.Fatal("environment obligation exposed requirement backing storage")
	}
	obligations[0] = emit.EnvironmentObligation{}
	if emission.EnvironmentObligations()[0].Identity() == "" {
		t.Fatal("environment emission exposed obligation backing storage")
	}
}

func environmentObligationOrder(
	left emit.EnvironmentObligation,
	right emit.EnvironmentObligation,
) int {
	if compared := strings.Compare(left.Identity(), right.Identity()); compared != 0 {
		return compared
	}
	if left.Kind() != right.Kind() {
		if left.Kind() < right.Kind() {
			return -1
		}
		return 1
	}
	return strings.Compare(left.TargetName(), right.TargetName())
}
