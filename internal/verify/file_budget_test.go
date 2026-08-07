package verify

import (
	"bufio"
	"fmt"
	"os"
	"testing"
)

const maximumDirectorySourceFiles = 20

type wallError struct {
	source string
	reason string
}

func (e *wallError) Error() string {
	return e.source + ": " + e.reason
}

func directoryFileBudgetError(
	directory string,
	sourceSet string,
	count int,
) error {
	if count <= maximumDirectorySourceFiles {
		return nil
	}
	return fmt.Errorf(
		"%s has %d maintained %s Go files, want at most %d",
		directory,
		count,
		sourceSet,
		maximumDirectorySourceFiles,
	)
}

func TestDirectoryFileBudgetMutationControls(t *testing.T) {
	for _, sourceSet := range []string{"production", "test"} {
		if err := directoryFileBudgetError(
			"internal/example",
			sourceSet,
			maximumDirectorySourceFiles,
		); err != nil {
			t.Fatalf("%s source set rejected at its exact budget: %v", sourceSet, err)
		}
		if err := directoryFileBudgetError(
			"internal/example",
			sourceSet,
			maximumDirectorySourceFiles+1,
		); err == nil {
			t.Fatalf("%s source set exceeded its budget without failure", sourceSet)
		}
	}
}

func physicalLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines, scanner.Err()
}
