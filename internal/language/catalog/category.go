package catalog

import "fmt"

// Category is the closed grammatical category of a construct Kind. The zero
// value CategoryInvalid is never valid; numCategories is the terminal sentinel.
type Category uint8

const (
	CategoryInvalid Category = iota
	CategoryExpression
	CategoryType
	CategoryStatement
	CategoryDeclaration
	CategorySpec
	CategoryStructural

	// numCategories is the terminal sentinel. It must remain last.
	numCategories
)

var categoryNames = [numCategories]string{
	CategoryExpression:  "expression",
	CategoryType:        "type",
	CategoryStatement:   "statement",
	CategoryDeclaration: "declaration",
	CategorySpec:        "spec",
	CategoryStructural:  "structural",
}

// Valid reports whether c names a category in the catalog.
func (c Category) Valid() bool { return c > CategoryInvalid && c < numCategories }

// String renders c for diagnostics.
func (c Category) String() string {
	if c.Valid() {
		return categoryNames[c]
	}
	return fmt.Sprintf("catalog.Category(%d)", uint8(c))
}
