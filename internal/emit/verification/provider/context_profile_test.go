package provider_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestContextProviderProfileConstructsExactInterfaceValue(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/contextprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package contextprofile

import (
	"context"
	"sync"
	"time"
)

type blockingFailure struct { mutex *sync.Mutex }

func (failure *blockingFailure) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "parent failed"
}

type fixedContext struct { failure error }

func (*fixedContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*fixedContext) Done() <-chan struct{} { return nil }
func (source *fixedContext) Err() error { return source.failure }
func (*fixedContext) Value(any) any { return nil }

func Run(key string, value string) (string, bool, string) {
	background := context.Background()
	todo := context.TODO()
	parent := &fixedContext{failure: &blockingFailure{mutex: &sync.Mutex{}}}
	child := context.WithValue(parent, key, value)
	selected, ok := child.Value(key).(string)
	identities := ok && child.Value("missing") == nil &&
		background == context.Background() && todo == context.TODO() &&
		background != todo
	return selected, identities, child.Err().Error()
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory:    project,
		Pattern:      ".",
		BuildProfile: linkedProviderBuildProfile(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = linkedProviderCertificate(t)
	// The context formatting family is implemented with exact chain
	// spellings: the constructor chain compiles through the used-provider
	// closure and emits the certified profile artifacts.
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(
			t,
			program.Roots()[0].Types().Scope().Lookup("Run"),
		)},
		options,
	)
	if err != nil {
		t.Fatalf("implemented context family failed the closure gate: %v", err)
	}
	if len(emission.Files()) == 0 {
		t.Fatal("context profile compilation emitted no target files")
	}
}
