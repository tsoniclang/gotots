// Package tokenize splits text into lowercase word tokens.
package tokenize

import (
	"strings"
	"unicode"
)

// Words yields the lowercase words of text as an iterator function.
func Words(text string) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		for _, field := range strings.FieldsFunc(text, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		}) {
			if !yield(strings.ToLower(field)) {
				return
			}
		}
	}
}

// Counts tallies word frequencies.
func Counts(text string) map[string]int {
	counts := map[string]int{}
	for word := range Words(text) {
		counts[word]++
	}
	return counts
}
