package emit_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestCooperativeDeferredRegistriesUseOneAwaitableCallableABI(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveNineConcurrencyDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("DeferCooperative"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{root},
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	printed := materializeArtifacts(t, emission, t.TempDir()).printed
	if err := verifyCooperativeDeferredRegistryProfiles(printed); err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(
		printed,
		"() => Awaitable<void>",
		"() => void",
		1,
	)
	if mutated == printed {
		t.Fatal("deferred-registry source-callable mutation was not applied")
	}
	if err := verifyCooperativeDeferredRegistryProfiles(mutated); err == nil {
		t.Fatal("synchronous source callable passed the cooperative registry gate")
	}
}

func verifyCooperativeDeferredRegistryProfiles(printed string) error {
	const marker = "new GoDeferredRegistry<"
	remaining := printed
	count := 0
	for {
		start := strings.Index(remaining, marker)
		if start < 0 {
			break
		}
		remaining = remaining[start+len(marker):]
		end := strings.Index(remaining, ";")
		if end < 0 {
			return fmt.Errorf("deferred-registry instantiation is unterminated")
		}
		profile := remaining[:end]
		if got := strings.Count(profile, "=> Awaitable<void>"); got != 3 {
			return fmt.Errorf(
				"deferred-registry callable profiles have %d Awaitable results, want 3: %s",
				got,
				profile,
			)
		}
		count++
		remaining = remaining[end+1:]
	}
	if count == 0 {
		return fmt.Errorf("cooperative deferred-registry instantiation is absent")
	}
	return nil
}
