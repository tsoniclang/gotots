package reflectvalue_test

import "testing"

// TestReflectMapOperationsCanonicalizeWithNativeEvidence covers MapRange
// iteration, SetMapIndex insertion and lookup, zero-Value results for
// missing keys, MakeMap over a MapOf descriptor composition, and
// SetIterKey/SetIterValue through addressable destinations. Every
// observation is independent of map iteration order, which Go itself
// randomizes.
func TestReflectMapOperationsCanonicalizeWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

func Merge() string {
	source := map[string]int{"alpha": 1, "beta": 2}
	target := map[string]int{"beta": 10}
	sourceValue := reflect.ValueOf(source)
	targetValue := reflect.ValueOf(target)
	iterator := sourceValue.MapRange()
	moved := 0
	for iterator.Next() {
		targetValue.SetMapIndex(iterator.Key(), iterator.Value())
		moved++
	}
	made := reflect.MakeMap(reflect.MapOf(reflect.TypeOf(""), reflect.TypeOf(0)))
	made.SetMapIndex(reflect.ValueOf("k"), reflect.ValueOf(5))
	missing := targetValue.MapIndex(reflect.ValueOf("gamma"))
	beta := targetValue.MapIndex(reflect.ValueOf("beta"))
	total := 0
	for _, value := range target {
		total += value
	}
	var lastKey string
	var lastCount int
	holderKey := reflect.ValueOf(&lastKey).Elem()
	holderCount := reflect.ValueOf(&lastCount).Elem()
	probe := sourceValue.MapRange()
	probe.Next()
	holderKey.SetIterKey(probe)
	holderCount.SetIterValue(probe)
	keyOK := lastKey == "alpha" || lastKey == "beta"
	countOK := lastCount == 1 || lastCount == 2
	return fmt.Sprintf(
		"%d %d %d %t %d %d %d %t %t %d",
		moved, len(target), total,
		missing.IsValid(), int(beta.Int()),
		made.Len(), int(made.MapIndex(reflect.ValueOf("k")).Int()),
		keyOK, countOK, sourceValue.Len(),
	)
}
`
	typescriptRunner := `const facts = Merge();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Merge())
}
`
	verifyReflectCanonical(
		t,
		source,
		"Merge",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}
