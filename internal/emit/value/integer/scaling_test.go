package integer_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestIntegerEmissionScalesWithUseSitesNotCarrierWidth(t *testing.T) {
	counts := []int{8, 16, 32}
	sourceBytes := make([]int, len(counts))
	targetBytes := make([]int, len(counts))
	targetFunctions := make([]int, len(counts))
	for index, count := range counts {
		source := scalingSource(count)
		sourceBytes[index] = len(source)
		directory := t.TempDir()
		writeFile(t, filepath.Join(directory, "go.mod"), `module example.com/integerscaling

go 1.26.4
`)
		writeFile(t, filepath.Join(directory, "source.go"), source)
		loaded, err := load.One(context.Background(), load.Request{
			Directory: directory,
			Pattern:   ".",
		})
		if err != nil {
			t.Fatal(err)
		}
		roots, err := emit.ExportedAPIRoots(loaded)
		if err != nil {
			t.Fatal(err)
		}
		emission, err := emit.Compile(loaded.Program(), roots)
		if err != nil {
			t.Fatal(err)
		}
		client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range emission.Files() {
			printed, printErr := client.PrintNode(
				file.SourceFile(),
				tsgo.PrintOptions{},
			)
			if printErr != nil {
				_ = client.Close()
				t.Fatal(printErr)
			}
			targetBytes[index] += len(printed)
			if file.Kind() != emit.TargetFileSource {
				continue
			}
			for _, statement := range file.SourceFile().Statements() {
				if _, ok := statement.(tsgo.FunctionDeclaration); ok {
					targetFunctions[index]++
				}
			}
		}
		if err := client.Close(); err != nil {
			t.Fatal(err)
		}
		if targetFunctions[index] != count {
			t.Fatalf(
				"%d sites emitted %d functions",
				count,
				targetFunctions[index],
			)
		}
	}
	for index := 1; index < len(counts); index++ {
		previousPerSite := float64(targetBytes[index-1]) /
			float64(counts[index-1])
		currentPerSite := float64(targetBytes[index]) /
			float64(counts[index])
		if currentPerSite > previousPerSite*1.05 {
			t.Fatalf(
				"target bytes/site grew %.2f -> %.2f",
				previousPerSite,
				currentPerSite,
			)
		}
	}
	t.Logf(
		"integer scaling sites=%v source-bytes=%v target-bytes=%v target-functions=%v",
		counts,
		sourceBytes,
		targetBytes,
		targetFunctions,
	)
}

func scalingSource(count int) string {
	types := []string{
		"int8",
		"int16",
		"int32",
		"int64",
		"uint8",
		"uint16",
		"uint32",
		"uint64",
	}
	var source strings.Builder
	source.WriteString("package integerscaling\n\n")
	for index := 0; index < count; index++ {
		sourceType := types[index%len(types)]
		fmt.Fprintf(
			&source,
			"func Value%03d(value %s) %s { return value + 1 }\n",
			index,
			sourceType,
			sourceType,
		)
	}
	return source.String()
}
