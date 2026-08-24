package sourceinvocation

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
)

type File struct {
	sourcePath string
	exact      bool
}

type Invocation struct {
	sourceIdentity         string
	sourcePath             string
	exportedName           string
	exactImplementation    bool
	inputParameters        []uint32
	resultOriginParameters []uint32
}

type Contract struct {
	files       []File
	invocations []Invocation
}

type ContractError struct {
	Reason string
}

func (e *ContractError) Error() string {
	return fmt.Sprintf("source invocation contract: %s", e.Reason)
}

func FromRuntime(
	assembled runtimeemission.Package,
) (Contract, error) {
	if !assembled.Valid() {
		return Contract{}, nil
	}
	selectedFiles := assembled.Files()
	files := make([]File, 0, len(selectedFiles))
	for _, file := range selectedFiles {
		files = append(files, File{
			sourcePath: file.OutputPath(),
			exact:      true,
		})
	}
	sort.Slice(files, func(left, right int) bool {
		return files[left].sourcePath < files[right].sourcePath
	})
	for index, file := range files {
		if file.sourcePath == "" {
			return Contract{}, contractError("source file path is empty")
		}
		if index != 0 && files[index-1].sourcePath == file.sourcePath {
			return Contract{}, contractError("source file path is duplicated")
		}
	}
	selectedInvocations := assembled.InvocationContracts()
	invocations := make([]Invocation, 0, len(selectedInvocations))
	for _, invocation := range selectedInvocations {
		invocations = append(invocations, Invocation{
			sourceIdentity:         invocation.SourceIdentity(),
			sourcePath:             invocation.SourcePath(),
			exportedName:           invocation.ExportedName(),
			exactImplementation:    invocation.ExactImplementation(),
			inputParameters:        invocation.InputParameters(),
			resultOriginParameters: invocation.ResultOriginParameters(),
		})
	}
	sort.Slice(invocations, func(left, right int) bool {
		if invocations[left].sourcePath != invocations[right].sourcePath {
			return invocations[left].sourcePath < invocations[right].sourcePath
		}
		return invocations[left].exportedName < invocations[right].exportedName
	})
	if err := validateInvocations(files, invocations); err != nil {
		return Contract{}, err
	}
	return Contract{files: files, invocations: invocations}, nil
}

func validateInvocations(files []File, invocations []Invocation) error {
	fileOwners := make(map[string]File, len(files))
	for _, file := range files {
		fileOwners[file.sourcePath] = file
	}
	identities := make(map[string]struct{}, len(invocations))
	for index, invocation := range invocations {
		file, found := fileOwners[invocation.sourcePath]
		if !found {
			return contractError("invocation has no source file owner")
		}
		if invocation.sourceIdentity == "" || invocation.exportedName == "" {
			return contractError("invocation identity is incomplete")
		}
		if file.exact && !invocation.exactImplementation {
			return contractError("exact file contains an inexact invocation")
		}
		if index != 0 && compareInvocation(invocations[index-1], invocation) == 0 {
			return contractError("source invocation is duplicated")
		}
		if _, duplicate := identities[invocation.sourceIdentity]; duplicate {
			return contractError("source identity is duplicated")
		}
		identities[invocation.sourceIdentity] = struct{}{}
	}
	return nil
}

func compareInvocation(left Invocation, right Invocation) int {
	if order := strings.Compare(left.sourcePath, right.sourcePath); order != 0 {
		return order
	}
	return strings.Compare(left.exportedName, right.exportedName)
}

func contractError(reason string) error {
	return &ContractError{Reason: reason}
}

func (c Contract) Valid() bool {
	return len(c.files) != 0
}

func (c Contract) Files() []File {
	return slices.Clone(c.files)
}

func (c Contract) Invocations() []Invocation {
	return slices.Clone(c.invocations)
}

func (f File) SourcePath() string {
	return f.sourcePath
}

func (f File) Exact() bool {
	return f.exact
}

func (c Invocation) SourceIdentity() string {
	return c.sourceIdentity
}

func (c Invocation) SourcePath() string {
	return c.sourcePath
}

func (c Invocation) ExportedName() string {
	return c.exportedName
}

func (c Invocation) ExactImplementation() bool {
	return c.exactImplementation
}

func (c Invocation) InputParameters() []uint32 {
	return slices.Clone(c.inputParameters)
}

func (c Invocation) ResultOriginParameters() []uint32 {
	return slices.Clone(c.resultOriginParameters)
}
