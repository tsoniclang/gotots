package semantic

import (
	"slices"
	"sort"
)

type canonicalKey[Self any] interface {
	Compare(Self) int
}

func sortCanonical[Record any, Key canonicalKey[Key]](
	records []Record,
	key func(Record) Key,
) {
	slices.SortFunc(records, func(left, right Record) int {
		return key(left).Compare(key(right))
	})
}

func searchCanonical[Record any, Key canonicalKey[Key]](
	records []Record,
	key func(Record) Key,
	target Key,
) int {
	return sort.Search(len(records), func(index int) bool {
		return key(records[index]).Compare(target) >= 0
	})
}
