package certify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPortableCooperativeSortHasOneBoundedWorkPath(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(
		repository,
		"gostdlib",
		"src",
		"internal",
		"portable",
		"slices",
		"sort.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	start := strings.Index(text, "export async function sortValues")
	if start < 0 {
		t.Fatal("portable cooperative sort owner is absent")
	}
	body := text[start:]
	for _, required := range []string{
		"for (let width = 1; width < values.length; width *= 2)",
		"await callComparison(compare, leftValue, rightValue)",
		"[source, target] = [target, source]",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("portable cooperative sort lacks %q", required)
		}
	}
	for _, forbidden := range []string{
		"await sortValues(",
		"[Symbol.iterator]",
		".slice(",
		".sort(",
		"instanceof Promise",
		"Promise.resolve",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("portable cooperative sort contains %q", forbidden)
		}
	}
}

func TestPortableSynchronousSortHasNoCooperativeDispatch(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(
		repository,
		"gostdlib",
		"src",
		"internal",
		"portable",
		"slices",
		"sort.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	body := string(source)
	start := strings.Index(body, "export function sortValuesSynchronous")
	if start < 0 {
		t.Fatal("portable synchronous sort owner is absent")
	}
	body = body[start:]
	for _, required := range []string{
		"for (let width = 1; width < values.length; width *= 2)",
		"compare(leftValue, rightValue)",
		"[source, target] = [target, source]",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("portable synchronous sort lacks %q", required)
		}
	}
	for _, forbidden := range []string{
		"await ",
		"callComparison(",
		"Promise",
		".sort(",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("portable synchronous sort contains %q", forbidden)
		}
	}
	if strings.Count(body, "sortValuesSynchronous") != 1 {
		t.Fatal("portable synchronous sort recurses or has duplicate owners")
	}
}
