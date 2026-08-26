package emit_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDeferredRegistriesUseOneSynchronousCallableABI(t *testing.T) {
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
	if err := verifySynchronousDeferredRegistryProfiles(printed); err != nil {
		t.Fatal(err)
	}
	mutated := mutateDeferredRegistryCallable(t, printed)
	if err := verifySynchronousDeferredRegistryProfiles(mutated); err == nil {
		t.Fatal("Promise-bearing source callable passed the synchronous registry gate")
	}
}

func mutateDeferredRegistryCallable(t *testing.T, printed string) string {
	t.Helper()
	const marker = "new GoDeferredRegistry<"
	profileStart := strings.Index(printed, marker)
	if profileStart < 0 {
		t.Fatal("deferred-registry mutation target is absent")
	}
	callableOffset := strings.Index(printed[profileStart:], "() => void")
	if callableOffset < 0 {
		t.Fatal("deferred-registry synchronous callable is absent")
	}
	callableStart := profileStart + callableOffset
	return printed[:callableStart] + "() => Promise<void>" +
		printed[callableStart+len("() => void"):]
}

func verifySynchronousDeferredRegistryProfiles(printed string) error {
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
		if got := strings.Count(profile, "=> void"); got != 3 {
			return fmt.Errorf(
				"deferred-registry callable profiles have %d void results, want 3: %s",
				got,
				profile,
			)
		}
		count++
		remaining = remaining[end+1:]
	}
	if count == 0 {
		return fmt.Errorf("synchronous deferred-registry instantiation is absent")
	}
	return nil
}
