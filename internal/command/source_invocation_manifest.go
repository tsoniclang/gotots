package command

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"slices"
	"strings"

	sourceinvocation "github.com/tsoniclang/gotots/internal/emit/sourceinvocation"
)

const sourceInvocationSchemaVersion = 3

type sourceInvocationDocument struct {
	SourceIdentity         string   `json:"sourceIdentity"`
	SourcePath             string   `json:"sourcePath"`
	ExportedName           string   `json:"export"`
	ExactImplementation    bool     `json:"exactImplementation"`
	InputParameters        []uint32 `json:"inputParameters"`
	ResultOriginParameters []uint32 `json:"resultOriginParameters"`
}

type sourceInvocationFileDocument struct {
	SourcePath   string `json:"sourcePath"`
	SourceDigest string `json:"sourceDigest"`
	Exact        bool   `json:"exact"`
}

type sourceInvocationContractDocument struct {
	SchemaVersion  int                            `json:"schemaVersion"`
	ContractDigest string                         `json:"contractDigest"`
	Files          []sourceInvocationFileDocument `json:"files"`
	Invocations    []sourceInvocationDocument     `json:"invocations"`
}

func encodeSourceInvocationContract(
	contract sourceinvocation.Contract,
	printedDigests map[string]string,
) (*sourceInvocationContractDocument, error) {
	if !contract.Valid() {
		return nil, nil
	}
	files := contract.Files()
	invocations := contract.Invocations()
	fileDocuments := make([]sourceInvocationFileDocument, 0, len(files))
	for _, file := range files {
		fileDocuments = append(fileDocuments, sourceInvocationFileDocument{
			SourcePath:   file.SourcePath(),
			SourceDigest: printedDigests[file.SourcePath()],
			Exact:        file.Exact(),
		})
	}
	documents := make([]sourceInvocationDocument, 0, len(invocations))
	for _, invocation := range invocations {
		documents = append(documents, sourceInvocationDocument{
			SourceIdentity:         invocation.SourceIdentity(),
			SourcePath:             invocation.SourcePath(),
			ExportedName:           invocation.ExportedName(),
			ExactImplementation:    invocation.ExactImplementation(),
			InputParameters:        cloneIndexes(invocation.InputParameters()),
			ResultOriginParameters: cloneIndexes(invocation.ResultOriginParameters()),
		})
	}
	return sealSourceInvocationContract(fileDocuments, documents)
}

func sealSourceInvocationContract(
	files []sourceInvocationFileDocument,
	invocations []sourceInvocationDocument,
) (*sourceInvocationContractDocument, error) {
	if len(files) == 0 {
		return nil, commandError(
			"encode source invocation contract",
			"source files are empty",
		)
	}
	fileOwners := make(map[string]sourceInvocationFileDocument, len(files))
	for index, file := range files {
		if !validSourceInvocationPath(file.SourcePath) {
			return nil, sourceInvocationError("source file path is invalid", index)
		}
		if !validDigest(file.SourceDigest) {
			return nil, sourceInvocationError("source file digest is invalid", index)
		}
		if index != 0 && files[index-1].SourcePath >= file.SourcePath {
			return nil, commandError(
				"encode source invocation contract",
				"source files are not strictly ordered",
			)
		}
		fileOwners[file.SourcePath] = file
	}
	identities := make(map[string]struct{}, len(invocations))
	for index, invocation := range invocations {
		file, found := fileOwners[invocation.SourcePath]
		if !found {
			return nil, sourceInvocationError("invocation has no source file owner", index)
		}
		if !validSemanticString(invocation.SourceIdentity) ||
			!validSemanticString(invocation.ExportedName) {
			return nil, sourceInvocationError("invocation identity is invalid", index)
		}
		if file.Exact && !invocation.ExactImplementation {
			return nil, sourceInvocationError(
				"exact source file has an inexact invocation implementation",
				index,
			)
		}
		if !invocation.ExactImplementation &&
			len(invocation.InputParameters) == 0 &&
			len(invocation.ResultOriginParameters) == 0 {
			return nil, sourceInvocationError("invocation carries no semantics", index)
		}
		if !strictlyOrderedIndexes(invocation.InputParameters) ||
			!strictlyOrderedIndexes(invocation.ResultOriginParameters) {
			return nil, sourceInvocationError("parameter indexes are not strictly ordered", index)
		}
		if index != 0 && compareSourceInvocation(
			invocations[index-1],
			invocation,
		) >= 0 {
			return nil, commandError(
				"encode source invocation contract",
				"invocations are not strictly ordered",
			)
		}
		if _, exists := identities[invocation.SourceIdentity]; exists {
			return nil, sourceInvocationError("source identity is duplicated", index)
		}
		identities[invocation.SourceIdentity] = struct{}{}
	}
	selectedFiles := slices.Clone(files)
	selectedInvocations := make([]sourceInvocationDocument, len(invocations))
	for index, invocation := range invocations {
		invocation.InputParameters = cloneIndexes(invocation.InputParameters)
		invocation.ResultOriginParameters = cloneIndexes(
			invocation.ResultOriginParameters,
		)
		selectedInvocations[index] = invocation
	}
	return &sourceInvocationContractDocument{
		SchemaVersion:  sourceInvocationSchemaVersion,
		ContractDigest: sourceInvocationDigest(selectedFiles, selectedInvocations),
		Files:          selectedFiles,
		Invocations:    selectedInvocations,
	}, nil
}

func sourceInvocationDigest(
	files []sourceInvocationFileDocument,
	invocations []sourceInvocationDocument,
) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte("gotots-source-invocation-v3\x00"))
	for _, file := range files {
		for _, value := range []string{file.SourcePath, file.SourceDigest} {
			_, _ = hash.Write([]byte(value))
			_, _ = hash.Write([]byte{0})
		}
		writeSourceInvocationFlag(hash, file.Exact)
	}
	for _, invocation := range invocations {
		for _, value := range []string{
			invocation.SourceIdentity,
			invocation.SourcePath,
			invocation.ExportedName,
		} {
			_, _ = hash.Write([]byte(value))
			_, _ = hash.Write([]byte{0})
		}
		writeSourceInvocationFlag(hash, invocation.ExactImplementation)
		writeSourceInvocationIndexes(hash, invocation.InputParameters)
		writeSourceInvocationIndexes(hash, invocation.ResultOriginParameters)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func writeSourceInvocationFlag(
	hash interface{ Write([]byte) (int, error) },
	selected bool,
) {
	value := byte(0)
	if selected {
		value = 1
	}
	_, _ = hash.Write([]byte{value})
}

func writeSourceInvocationIndexes(
	hash interface{ Write([]byte) (int, error) },
	indexes []uint32,
) {
	for _, index := range indexes {
		_, _ = fmt.Fprintf(hash, "%d,", index)
	}
	_, _ = hash.Write([]byte{0})
}

func cloneIndexes(indexes []uint32) []uint32 {
	result := make([]uint32, len(indexes))
	copy(result, indexes)
	return result
}

func strictlyOrderedIndexes(indexes []uint32) bool {
	for index := 1; index < len(indexes); index++ {
		if indexes[index-1] >= indexes[index] {
			return false
		}
	}
	return true
}

func compareSourceInvocation(
	left sourceInvocationDocument,
	right sourceInvocationDocument,
) int {
	if order := strings.Compare(left.SourcePath, right.SourcePath); order != 0 {
		return order
	}
	return strings.Compare(left.ExportedName, right.ExportedName)
}

func validSourceInvocationPath(value string) bool {
	return value != "" &&
		!strings.Contains(value, "\\") &&
		path.Clean(value) == value &&
		value != "." &&
		value != ".." &&
		!strings.HasPrefix(value, "../") &&
		!strings.HasPrefix(value, "/") &&
		strings.HasSuffix(value, ".ts") &&
		!strings.HasSuffix(value, ".d.ts")
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validSemanticString(value string) bool {
	return value != "" && !strings.ContainsRune(value, 0)
}

func sourceInvocationError(reason string, index int) error {
	return commandError(
		"encode source invocation contract",
		fmt.Sprintf("row %d: %s", index, reason),
	)
}
