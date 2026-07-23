package stagecheck

import (
	"fmt"
	"reflect"
	"testing"
)

func TestProblemSetIsBoundedAndOrderIndependent(t *testing.T) {
	first := newProblemSet()
	second := newProblemSet()
	for index := 99; index >= 0; index-- {
		first.add(fmt.Sprintf("residual-%03d", index))
	}
	for index := 0; index < 100; index++ {
		second.add(fmt.Sprintf("residual-%03d", index))
	}
	if first.count != 100 || second.count != 100 {
		t.Fatalf("residual counts = %d/%d", first.count, second.count)
	}
	if len(first.samples) != diagnosticSampleLimit ||
		len(second.samples) != diagnosticSampleLimit {
		t.Fatalf("sample sizes = %d/%d", len(first.samples), len(second.samples))
	}
	if first.digest() != second.digest() ||
		!reflect.DeepEqual(first.samples, second.samples) {
		t.Fatalf(
			"order changed diagnostics:\nfirst=%s %v\nsecond=%s %v",
			first.digest(), first.samples, second.digest(), second.samples,
		)
	}
	first.add("residual-000")
	if first.count != 101 || first.digest() == second.digest() {
		t.Fatal("multiset digest did not account for a duplicate residual")
	}
}
