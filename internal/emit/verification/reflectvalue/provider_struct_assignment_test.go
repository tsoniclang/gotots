package reflectvalue_test

import (
	"regexp"
	"strings"
	"testing"
)

func TestReflectProviderStructAssignmentCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"bufio"
	"encoding/base32"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
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

	regexpSource := regexp.MustCompile("^source$")
	regexpTarget := regexp.MustCompile("^target$")
	*regexpTarget = *regexpSource
	regexpAssigned := regexpTarget.MatchString("source") &&
		!regexpTarget.MatchString("target")
	urlSource, _ := url.Parse("https://source.example/path?q=1")
	urlTarget, _ := url.Parse("https://target.example/old")
	*urlTarget = *urlSource
	urlAssigned := urlTarget.Scheme == "https" &&
		urlTarget.Host == "source.example" &&
		urlTarget.Path == "/path" &&
		urlTarget.RawQuery == "q=1"
	base32Source := *base32.StdEncoding
	base32Target := *base32.HexEncoding
	base32Target = base32Source
	base32Assigned := string(base32Target.AppendEncode(nil, []byte{0xff, 0xef})) ==
		"77XQ====" &&
		string(base32.HexEncoding.AppendEncode(nil, []byte{0xff, 0xef})) ==
			"VVNG===="
	replacerSource := *strings.NewReplacer("a", "source")
	replacerTarget := *strings.NewReplacer("a", "target")
	replacerTarget = replacerSource
	replacerAssigned := replacerTarget.Replace("a") == "source"
	scannerSource := bufio.NewScanner(strings.NewReader("one\ntwo\n"))
	scannerTarget := bufio.NewScanner(strings.NewReader("target\n"))
	*scannerTarget = *scannerSource
	scannerTargetFirst := scannerTarget.Scan() && scannerTarget.Text() == "one"
	scannerSourceExhausted := !scannerSource.Scan()
	scannerTargetSecond := scannerTarget.Scan() && scannerTarget.Text() == "two"

	return fmt.Sprintf(
		"parse=%q/%q/%q/%q/%q mutex=%t builder=%q/%t/%t/%t/%q timer=%t/%t/%t/%t/%t/%t/%t regexp=%t url=%t base32=%t replacer=%t scanner=%t/%t/%t",
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
		regexpAssigned,
		urlAssigned,
		base32Assigned,
		replacerAssigned,
		scannerTargetFirst,
		scannerSourceExhausted,
		scannerTargetSecond,
	)
}
`
	typescriptRunner := `const facts = ProviderAssignments();
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
			if !strings.Contains(
				artifacts.printed,
				"ReflectTypeMetadataOperations.$registerOpaqueStruct(",
			) {
				t.Fatalf("provider struct reflection lacks its compact opaque registration")
			}
			if !regexp.MustCompile(
				`(?s)\.\$registerOpaqueStruct\(\s*[^,]+,\s*\(\)\s*=>`,
			).MatchString(artifacts.printed) {
				t.Fatalf("provider struct reflection resolves its adapter eagerly")
			}
			for _, required := range []string{
				"TimeParseErrorOperations.$copy",
				"TimeTimerOperations.$copy",
				"SyncMutexOperations.$copy",
				"StringsBuilderOperations.$copy",
				"RegexpValueOperations.$assign",
				"NetUrlURLOperations.$assign",
				"Base32EncodingOperations.$assign",
				"StringsReplacerOperations.$assign",
			} {
				if !strings.Contains(artifacts.printed, required) {
					relevant := make([]string, 0)
					for _, line := range strings.Split(artifacts.printed, "\n") {
						if strings.Contains(line, "ParseError") ||
							strings.Contains(line, "GoProviderState") {
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
			if !regexp.MustCompile(
				`GoProviderState\.\$assign`,
			).MatchString(artifacts.printed) ||
				!strings.Contains(artifacts.printed, "DirectBufioScanner") {
				t.Fatalf("scanner assignment lacks its direct provider state operation")
			}
		},
	)
}
