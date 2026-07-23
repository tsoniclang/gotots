package semantic

import "sort"

type canonicalRecords[Record any] struct {
	records []Record
	keys    []string
}

func (records canonicalRecords[Record]) Len() int {
	return len(records.records)
}

func (records canonicalRecords[Record]) Less(left, right int) bool {
	return records.keys[left] < records.keys[right]
}

func (records canonicalRecords[Record]) Swap(left, right int) {
	records.records[left], records.records[right] =
		records.records[right], records.records[left]
	records.keys[left], records.keys[right] =
		records.keys[right], records.keys[left]
}

func sortCanonical[Record any](
	records []Record,
	key func(Record) string,
) {
	if len(records) < 2 {
		return
	}
	keys := make([]string, len(records))
	for index, record := range records {
		keys[index] = key(record)
	}
	sort.Sort(canonicalRecords[Record]{
		records: records,
		keys:    keys,
	})
}

func searchCanonical(
	length int,
	key func(int) string,
	target string,
) int {
	return sort.Search(length, func(index int) bool {
		return key(index) >= target
	})
}
