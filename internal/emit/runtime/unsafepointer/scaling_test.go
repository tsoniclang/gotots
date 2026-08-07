package unsafepointer

import (
	"math/bits"
	"testing"
)

func TestIndexedAddressLookupHasLogarithmicComparisonBound(t *testing.T) {
	for count := 1; count <= 4096; count *= 2 {
		bound := bits.Len(uint(count)) + 1
		for target := range count {
			comparisons := indexedLookupComparisons(count, target)
			if comparisons > bound {
				t.Fatalf(
					"allocations=%d target=%d comparisons=%d, bound=%d",
					count,
					target,
					comparisons,
					bound,
				)
			}
		}
	}
	if comparisons, bound := 4096, bits.Len(uint(4096))+1; comparisons <= bound {
		t.Fatalf("linear-scan control comparisons=%d, bound=%d", comparisons, bound)
	}
}

func TestRegionWorkDependsOnTouchedRangeNotBackingLength(t *testing.T) {
	for _, testCase := range []struct {
		offset int
		length int
		size   int
		want   int
	}{
		{offset: 0, length: 1, size: 4, want: 1},
		{offset: 3, length: 2, size: 4, want: 2},
		{offset: 4, length: 8, size: 4, want: 2},
		{offset: 7, length: 9, size: 4, want: 3},
	} {
		for _, backingLength := range []int{8, 1 << 20} {
			actual := touchedElementCount(
				testCase.offset,
				testCase.length,
				testCase.size,
			)
			if actual != testCase.want || actual >= backingLength {
				t.Fatalf(
					"offset=%d length=%d size=%d backing=%d touched=%d, want=%d",
					testCase.offset,
					testCase.length,
					testCase.size,
					backingLength,
					actual,
					testCase.want,
				)
			}
		}
	}
}

func indexedLookupComparisons(count int, target int) int {
	low := 0
	high := count - 1
	comparisons := 0
	for low <= high {
		comparisons++
		middle := (low + high) / 2
		if middle <= target {
			low = middle + 1
		} else {
			high = middle - 1
		}
	}
	return comparisons
}

func touchedElementCount(offset int, length int, size int) int {
	first := offset / size
	last := (offset + length + size - 1) / size
	return last - first
}
