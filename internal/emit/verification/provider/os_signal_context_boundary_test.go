package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestNotifyContextPreservesCanonicalContextAndError(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/signalcontext\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package signalcontext

import (
	"context"
	"os"
	"os/signal"
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

func Result() (bool, string, string) {
	parent := &fixedContext{
		failure: &blockingFailure{mutex: new(sync.Mutex)},
	}
	parentMessage := parent.Err().Error()
	child, stop := signal.NotifyContext(parent, os.Interrupt)
	stop()
	childMessage := ""
	if failure := child.Err(); failure != nil {
		childMessage = failure.Error()
	}
	return child != nil, parentMessage, childMessage
}

var _ context.Context = (*fixedContext)(nil)
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
	// signal.NotifyContext reaches the certified context value-formatting
	// placeholder through its context construction chain; the used-provider
	// closure must fail this compilation before any target file is sealed.
	// The runtime differential returns when the context formatting family
	// is implemented.
	_, err = emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(
			t,
			program.Roots()[0].Types().Scope().Lookup("Result"),
		)},
		options,
	)
	if err == nil {
		t.Fatal("used context formatting placeholder passed the closure gate")
	}
	if !strings.Contains(err.Error(), "used provider placeholders") ||
		!strings.Contains(err.Error(), "ContextValue") {
		t.Fatalf("closure diagnostic = %v", err)
	}
}
