package command

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSourceInvocationContractSealsCanonicalRows(t *testing.T) {
	files, invocations := sourceInvocationFixture()
	contract, err := sealSourceInvocationContract(files, invocations)
	if err != nil {
		t.Fatal(err)
	}
	if contract.SchemaVersion != sourceInvocationSchemaVersion {
		t.Fatalf("schema version = %d", contract.SchemaVersion)
	}
	const expectedDigest = "d775e19d7c0fc42b7f7dc0b5c82248460b63ba2be507efa9d3d1b9342c453c61"
	if contract.ContractDigest != expectedDigest {
		t.Fatalf("contract digest = %q, want %q", contract.ContractDigest, expectedDigest)
	}
	payload, err := json.Marshal(contract)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`"inputParameters":[]`,
		`"resultOriginParameters":[]`,
	} {
		if !strings.Contains(string(payload), expected) {
			t.Fatalf("source invocation payload has no %s: %s", expected, payload)
		}
	}
	invocations[0].ResultOriginParameters[0] = 9
	if contract.Invocations[0].ResultOriginParameters[0] != 0 {
		t.Fatal("sealed source invocation contract exposes input index storage")
	}
}

func TestSourceInvocationContractRejectsInvalidRows(t *testing.T) {
	tests := []struct {
		name   string
		mutate func([]sourceInvocationFileDocument, []sourceInvocationDocument) (
			[]sourceInvocationFileDocument,
			[]sourceInvocationDocument,
		)
	}{
		{
			name: "empty files",
			mutate: func(_ []sourceInvocationFileDocument, invocations []sourceInvocationDocument) (
				[]sourceInvocationFileDocument,
				[]sourceInvocationDocument,
			) {
				return nil, invocations
			},
		},
		{
			name: "invalid source path",
			mutate: func(files []sourceInvocationFileDocument, invocations []sourceInvocationDocument) (
				[]sourceInvocationFileDocument,
				[]sourceInvocationDocument,
			) {
				files[0].SourcePath = "../runtime/a.ts"
				return files, invocations
			},
		},
		{
			name: "invalid digest",
			mutate: func(files []sourceInvocationFileDocument, invocations []sourceInvocationDocument) (
				[]sourceInvocationFileDocument,
				[]sourceInvocationDocument,
			) {
				files[0].SourceDigest = "not-a-digest"
				return files, invocations
			},
		},
		{
			name: "reordered files",
			mutate: func(files []sourceInvocationFileDocument, invocations []sourceInvocationDocument) (
				[]sourceInvocationFileDocument,
				[]sourceInvocationDocument,
			) {
				files[0], files[1] = files[1], files[0]
				return files, invocations
			},
		},
		{
			name: "orphan invocation",
			mutate: func(files []sourceInvocationFileDocument, invocations []sourceInvocationDocument) (
				[]sourceInvocationFileDocument,
				[]sourceInvocationDocument,
			) {
				invocations[0].SourcePath = "runtime/missing.ts"
				return files, invocations
			},
		},
		{
			name: "inexact implementation in exact file",
			mutate: func(files []sourceInvocationFileDocument, invocations []sourceInvocationDocument) (
				[]sourceInvocationFileDocument,
				[]sourceInvocationDocument,
			) {
				invocations[0].ExactImplementation = false
				return files, invocations
			},
		},
		{
			name: "empty invocation semantics",
			mutate: func(files []sourceInvocationFileDocument, invocations []sourceInvocationDocument) (
				[]sourceInvocationFileDocument,
				[]sourceInvocationDocument,
			) {
				invocations[1].InputParameters = nil
				return files, invocations
			},
		},
		{
			name: "unordered parameter indexes",
			mutate: func(files []sourceInvocationFileDocument, invocations []sourceInvocationDocument) (
				[]sourceInvocationFileDocument,
				[]sourceInvocationDocument,
			) {
				invocations[1].InputParameters = []uint32{2, 0}
				return files, invocations
			},
		},
		{
			name: "reordered invocations",
			mutate: func(files []sourceInvocationFileDocument, invocations []sourceInvocationDocument) (
				[]sourceInvocationFileDocument,
				[]sourceInvocationDocument,
			) {
				invocations[0], invocations[1] = invocations[1], invocations[0]
				return files, invocations
			},
		},
		{
			name: "duplicate source identity",
			mutate: func(files []sourceInvocationFileDocument, invocations []sourceInvocationDocument) (
				[]sourceInvocationFileDocument,
				[]sourceInvocationDocument,
			) {
				invocations[1].SourceIdentity = invocations[0].SourceIdentity
				return files, invocations
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			files, invocations := sourceInvocationFixture()
			files, invocations = test.mutate(files, invocations)
			if _, err := sealSourceInvocationContract(files, invocations); err == nil {
				t.Fatal("invalid source invocation contract was accepted")
			}
		})
	}
}

func sourceInvocationFixture() (
	[]sourceInvocationFileDocument,
	[]sourceInvocationDocument,
) {
	return []sourceInvocationFileDocument{
			{SourcePath: "runtime/a.ts", SourceDigest: strings.Repeat("a", 64), Exact: true},
			{SourcePath: "runtime/b.ts", SourceDigest: strings.Repeat("b", 64), Exact: false},
		}, []sourceInvocationDocument{
			{
				SourceIdentity:         "runtime:a",
				SourcePath:             "runtime/a.ts",
				ExportedName:           "Alpha",
				ExactImplementation:    true,
				InputParameters:        []uint32{},
				ResultOriginParameters: []uint32{0},
			},
			{
				SourceIdentity:         "runtime:b",
				SourcePath:             "runtime/b.ts",
				ExportedName:           "Beta",
				ExactImplementation:    false,
				InputParameters:        []uint32{0, 2},
				ResultOriginParameters: []uint32{},
			},
		}
}
