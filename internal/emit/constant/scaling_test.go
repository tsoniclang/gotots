package constant_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

// TestLargeConstantProjectionScalesWithUsesNotValueSize proves the projection
// design is O(value-size + uses), never O(value-size × uses): a large untyped
// string constant used 1, 2, and 4 times materializes its payload once in the
// executable projection and once in its canonical source fact. Every use is a
// constant-size reference. If uses inlined the value, the payload count would
// grow with uses and the artifact would grow by the payload size per use.
func TestLargeConstantProjectionScalesWithUsesNotValueSize(t *testing.T) {
	const payload = "PAYLOAD_" // repeated to a large, distinctive literal
	large := strings.Repeat(payload, 512)

	sizes := make(map[int]int)
	for _, uses := range []int{1, 2, 4} {
		printed := compileScalingProgram(t, large, uses)

		if count := strings.Count(printed, large); count != 2 {
			t.Fatalf("uses=%d: payload appears %d times, want projection plus source fact", uses, count)
		}
		// Every use is a constant-size reference to the projection.
		if refs := strings.Count(printed, "Banner$string"); refs < uses {
			t.Fatalf("uses=%d: found %d Banner$string references, want at least %d", uses, refs, uses)
		}
		sizes[uses] = len(printed)
	}

	// Going from 1 to 4 uses adds three references, not three payload copies.
	// The growth must be a tiny fraction of a single payload — proof the payload
	// is not duplicated per use.
	growth := sizes[4] - sizes[1]
	if growth >= len(large) {
		t.Fatalf("artifact grew by %d bytes across 3 added uses (payload is %d bytes); uses are not constant-size",
			growth, len(large))
	}
}

// TestLargeDefinedConstantThunkScalesWithUsesNotValueSize proves the ESM-safe
// defined-basic path also materializes a large payload once in executable code
// and once in its source fact. Added uses invoke the same typed thunk and never
// duplicate either payload.
func TestLargeDefinedConstantThunkScalesWithUsesNotValueSize(t *testing.T) {
	const payload = "DEFINED_PAYLOAD_"
	large := strings.Repeat(payload, 512)

	sizes := make(map[int]int)
	for _, uses := range []int{1, 2, 4} {
		printed := compileDefinedScalingProgram(t, large, uses)
		if count := strings.Count(printed, large); count != 2 {
			t.Fatalf(
				"uses=%d: defined payload appears %d times, want thunk plus source fact",
				uses,
				count,
			)
		}
		if calls := strings.Count(printed, "Banner$constant()"); calls < uses {
			t.Fatalf(
				"uses=%d: found %d Banner$constant calls, want at least %d",
				uses,
				calls,
				uses,
			)
		}
		sizes[uses] = len(printed)
	}

	growth := sizes[4] - sizes[1]
	if growth >= len(large) {
		t.Fatalf(
			"defined artifact grew by %d bytes across 3 added uses (payload is %d bytes)",
			growth,
			len(large),
		)
	}
}

func compileScalingProgram(t *testing.T, payload string, uses int) string {
	t.Helper()
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "go.mod"), "module example.com/scaling\n\ngo 1.26.4\n")

	var builder strings.Builder
	builder.WriteString("package scaling\n\n")
	fmt.Fprintf(&builder, "const Banner = %q\n\n", payload)
	// Separate functions keep each use out of a constant-foldable expression, so
	// each is a genuine independent reference to the projection.
	for index := 0; index < uses; index++ {
		fmt.Fprintf(&builder, "func Use%d() string {\n\treturn Banner\n}\n\n", index)
	}
	writeFile(t, filepath.Join(directory, "source.go"), builder.String())

	loaded, err := load.One(context.Background(), load.Request{Directory: directory, Pattern: "."})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatalf("scaling compile (uses=%d) failed: %v", uses, err)
	}
	return printConstantFamily(t, emission)
}

func compileDefinedScalingProgram(t *testing.T, payload string, uses int) string {
	t.Helper()
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "go.mod"), "module example.com/definedscaling\n\ngo 1.26.4\n")

	var builder strings.Builder
	builder.WriteString("package definedscaling\n\ntype Label string\n\n")
	fmt.Fprintf(&builder, "const Banner Label = %q\n\n", payload)
	for index := 0; index < uses; index++ {
		fmt.Fprintf(&builder, "func Use%d() Label {\n\treturn Banner\n}\n\n", index)
	}
	writeFile(t, filepath.Join(directory, "source.go"), builder.String())

	loaded, err := load.One(context.Background(), load.Request{Directory: directory, Pattern: "."})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatalf("defined scaling compile (uses=%d) failed: %v", uses, err)
	}
	return printConstantFamily(t, emission)
}
