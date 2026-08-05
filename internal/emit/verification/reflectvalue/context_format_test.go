package reflectvalue_test

import "testing"

// TestContextFormattingMatchesGo proves the context chain formatting
// family: Background, TODO, and WithCancel chains print their exact Go
// spellings through the ordinary fmt path.
func TestContextFormattingMatchesGo(t *testing.T) {
	source := `package reflectvalue

import (
	"context"
	"fmt"
)

func Describe() string {
	base := context.Background()
	child, cancel := context.WithCancel(base)
	grandchild, nestedCancel := context.WithCancel(child)
	described := fmt.Sprintf(
		"%v | %v | %v | %v",
		base, context.TODO(), child, grandchild,
	)
	nestedCancel()
	cancel()
	return described
}
`
	typescriptRunner := `const facts = await Describe();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Describe())
}
`
	runReflectDifferential(
		t,
		source,
		"Describe",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}
