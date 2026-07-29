package verify

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func closureDirectory(relative string) string {
	return filepath.Join(
		"..",
		"..",
		"testdata",
		"constructs",
		"closure",
		"wave10",
		relative,
	)
}

func loadClosurePackage(
	t *testing.T,
	relative string,
) (*load.Program, *load.Package) {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: closureDirectory(relative),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return program, program.Roots()[0]
}

func closureRoot(
	t *testing.T,
	sourcePackage *load.Package,
	name string,
) emit.Root {
	t.Helper()
	root, err := emit.NewRoot(sourcePackage.Types().Scope().Lookup(name))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

type closureArtifacts struct {
	files   int
	bytes   int
	nodes   int
	largest int
	printed string
}

func materializeClosure(
	t *testing.T,
	emission emit.ProgramEmission,
) closureArtifacts {
	t.Helper()
	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(t), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	result := closureArtifacts{}
	var targetPaths []string
	var printed strings.Builder
	for _, file := range emission.Files() {
		target, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{
			": any",
			": unknown",
			" as any",
			" as unknown",
			".call(",
			".apply(",
			".bind(",
			"import(",
		} {
			if strings.Contains(target, forbidden) {
				t.Fatalf("%s contains forbidden %q", file.OutputPath(), forbidden)
			}
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(targetPath, []byte(target), 0o644); err != nil {
			t.Fatal(err)
		}
		targetPaths = append(targetPaths, targetPath)
		result.files++
		result.bytes += len(target)
		result.nodes += encodedClosureNodes(t, encoded)
		if len(target) > result.largest {
			result.largest = len(target)
		}
		printed.WriteString(target)
		printed.WriteByte('\n')
	}
	if result.files == 0 ||
		result.bytes > 350_000 ||
		result.nodes > 60_000 ||
		result.largest > 80_000 {
		t.Fatalf(
			"Wave 10 artifact bounds: files=%d bytes=%d nodes=%d largest=%d",
			result.files,
			result.bytes,
			result.nodes,
			result.largest,
		)
	}
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noEmit",
	}
	arguments = append(arguments, targetPaths...)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(t),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	result.printed = printed.String()
	return result
}

func encodedClosureNodes(t *testing.T, encoded []byte) int {
	t.Helper()
	const (
		headerSize       = 44
		nodesOffsetField = 40
		nodeWidth        = 28
	)
	if len(encoded) < headerSize {
		t.Fatalf("encoded target is %d bytes, want protocol header", len(encoded))
	}
	nodesOffset := int(binary.LittleEndian.Uint32(
		encoded[nodesOffsetField:headerSize],
	))
	if nodesOffset < headerSize ||
		nodesOffset > len(encoded) ||
		(len(encoded)-nodesOffset)%nodeWidth != 0 {
		t.Fatalf("encoded target has invalid node offset %d", nodesOffset)
	}
	return (len(encoded) - nodesOffset) / nodeWidth
}
