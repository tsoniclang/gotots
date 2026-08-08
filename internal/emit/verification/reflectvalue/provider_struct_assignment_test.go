package reflectvalue_test

import (
	"strings"
	"testing"
)

func TestReflectProviderStructAssignmentCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

func panics(action func()) (result bool) {
	defer func() {
		result = recover() != nil
	}()
	action()
	return false
}

func stopTimer(timer *time.Timer) (stopped bool, panicked bool) {
	defer func() {
		panicked = recover() != nil
	}()
	stopped = timer.Stop()
	return stopped, false
}

func ProviderAssignments() string {
	parseSource := &time.ParseError{
		Layout:     "layout",
		Value:      "value",
		LayoutElem: "layout-elem",
		ValueElem:  "value-elem",
		Message:    "message",
	}
	parseTarget := &time.ParseError{}
	reflect.ValueOf(parseTarget).Elem().Set(
		reflect.ValueOf(parseSource).Elem(),
	)

	mutexSource := &sync.Mutex{}
	mutexTarget := &sync.Mutex{}
	reflect.ValueOf(mutexTarget).Elem().Set(
		reflect.ValueOf(mutexSource).Elem(),
	)
	mutexPanicked := panics(func() {
		mutexTarget.Lock()
		mutexTarget.Unlock()
	})

	builderSource := &strings.Builder{}
	_, _ = builderSource.WriteString("source")
	builderTarget := &strings.Builder{}
	reflect.ValueOf(builderTarget).Elem().Set(
		reflect.ValueOf(builderSource).Elem(),
	)
	builderText := builderTarget.String()
	builderCopyPanicked := panics(func() {
		_, _ = builderTarget.WriteString("-copy")
	})
	reflect.ValueOf(builderSource).Elem().Set(
		reflect.ValueOf(builderSource).Elem(),
	)
	builderSelfPanicked := panics(func() {
		_, _ = builderSource.WriteString("-self")
	})
	builderZeroSource := &strings.Builder{}
	builderZeroTarget := &strings.Builder{}
	reflect.ValueOf(builderZeroTarget).Elem().Set(
		reflect.ValueOf(builderZeroSource).Elem(),
	)
	builderZeroPanicked := panics(func() {
		_, _ = builderZeroTarget.WriteString("zero")
	})

	timerSource := time.NewTimer(time.Hour)
	timerTarget := time.NewTimer(time.Hour)
	_ = timerTarget.Stop()
	reflect.ValueOf(timerTarget).Elem().Set(
		reflect.ValueOf(timerSource).Elem(),
	)
	timerSharesChannel := timerTarget.C == timerSource.C
	timerTargetStopped, timerTargetPanicked := stopTimer(timerTarget)
	timerActiveTarget := time.NewTimer(time.Hour)
	reflect.ValueOf(timerActiveTarget).Elem().Set(
		reflect.ValueOf(timerSource).Elem(),
	)
	timerActiveTargetStopped, timerActiveTargetPanicked := stopTimer(timerActiveTarget)
	timerSourceStopped, timerSourcePanicked := stopTimer(timerSource)

	return fmt.Sprintf(
		"parse=%q/%q/%q/%q/%q mutex=%t builder=%q/%t/%t/%t/%q timer=%t/%t/%t/%t/%t/%t/%t",
		parseTarget.Layout,
		parseTarget.Value,
		parseTarget.LayoutElem,
		parseTarget.ValueElem,
		parseTarget.Message,
		mutexPanicked,
		builderText,
		builderCopyPanicked,
		builderSelfPanicked,
		builderZeroPanicked,
		builderSource.String(),
		timerSharesChannel,
		timerTargetStopped,
		timerTargetPanicked,
		timerActiveTargetStopped,
		timerActiveTargetPanicked,
		timerSourceStopped,
		timerSourcePanicked,
	)
}
`
	typescriptRunner := `const facts = await ProviderAssignments();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.ProviderAssignments())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"ProviderAssignments",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			for _, required := range []string{
				"TimeParseErrorOperations.$copy",
				"TimeTimerOperations.$copy",
				"SyncMutexOperations.$copy",
				"StringsBuilderOperations.$copy",
			} {
				if !strings.Contains(artifacts.printed, required) {
					relevant := make([]string, 0)
					for _, line := range strings.Split(artifacts.printed, "\n") {
						if strings.Contains(line, "ParseError") ||
							strings.Contains(line, "$goProviderState_") {
							relevant = append(relevant, line)
						}
					}
					t.Fatalf(
						"provider assignment artifact lacks %q:\n%s",
						required,
						strings.Join(relevant, "\n"),
					)
				}
			}
		},
	)
}
