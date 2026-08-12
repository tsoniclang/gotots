package sourceimplementation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

func TestVerifySelectsCanonicalPackageAndOfficialSourceFile(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "go.mod"), "module example.test/app\n\ngo 1.26.4\n")
	writeFixture(t, filepath.Join(root, "fast", "fast.go"), `package fast

type Digest struct { Hi uint64; Lo uint64 }
func Sum(value string) Digest { return Digest{Hi: uint64(len(value))} }
`)
	writeFixture(t, filepath.Join(root, "main.go"), `package main
import "example.test/app/fast"
func main() { _ = fast.Sum("value") }
`)
	implementation := filepath.Join(root, "implementation")
	writeFixture(t, filepath.Join(implementation, "package.ts"), `export type Digest$Storage = { Hi: bigint; Lo: bigint };
export class Digest {}
export function Sum(value: string): Digest { return new Digest(); }
export function $initialize(): void {}
`)
	writeFixture(t, filepath.Join(implementation, "private.ts"), `import type { Digest } from "./package.js";
export type DigestView = { value: Digest };
`)
	writeFixture(t, filepath.Join(implementation, "tsconfig.json"), `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true,
    "noEmit": true
  },
	  "files": ["package.ts", "private.ts"]
}
`)
	repository, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	selectedGo, err := toolchain.ResolveGo(
		"",
		filepath.Join(repository, ".temp", "cache", "toolchain-tests"),
	)
	if err != nil {
		t.Fatal(err)
	}
	selectedTSGo, err := tsgo.ResolveTool(selectedGo, repository, "")
	if err != nil {
		t.Fatal(err)
	}
	contract := Document{
		SchemaVersion: SchemaVersion,
		Package: PackageDocument{
			ImportPath:    "example.test/app/fast",
			ModulePath:    "example.test/app",
			ModuleVersion: "",
		},
		Build: BuildDocument{
			GoVersion:  selectedGo.Version(),
			GOOS:       selectedGo.DefaultGOOS(),
			GOARCH:     selectedGo.DefaultGOARCH(),
			CGOEnabled: false,
		},
		Compilation: CompilationDocument{
			Integers: "number", EvaluationOrder: "direct", Concurrency: "disabled",
		},
		Source:   "package.ts",
		TSConfig: "tsconfig.json",
		Envelope: EnvelopeDocument{Kind: EnvelopeExact},
		Exports:  []string{"$initialize", "Digest", "Digest$Storage", "Sum"},
		PrivateModules: []PrivateModuleDocument{{
			GoFile: "fast.go", Source: "private.ts", Exports: []string{"DigestView"},
		}},
	}
	payload, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join(implementation, "contract.json")
	writeFixture(t, contractPath, string(payload)+"\n")
	program, err := load.Load(context.Background(), load.Request{
		Directory: root,
		Pattern:   ".",
		GoTool:    selectedGo,
	})
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := VerifyAll(Config{
		RepositoryRoot: repository,
		Program:        program,
		ContractPaths:  []string{contractPath},
		ScratchRoot:    filepath.Join(root, ".scratch"),
		Compilation: CompilationDocument{
			Integers: "number", EvaluationOrder: "direct", Concurrency: "disabled",
		},
		TSGoTool: selectedTSGo,
	})
	if err != nil {
		t.Fatal(err)
	}
	selected := program.PackageByPath("example.test/app/fast")
	implementationRecord, ok := certificate.ForPackage(selected)
	if !ok {
		t.Fatal("canonical source package has no implementation")
	}
	if implementationRecord.Digest() == "" ||
		implementationRecord.SourceFile() == nil ||
		implementationRecord.Envelope() != EnvelopeExact {
		t.Fatal("verified implementation evidence is incomplete")
	}
	privateModules := implementationRecord.PrivateModules()
	if len(privateModules) != 1 || privateModules[0].GoFile() != "fast.go" ||
		privateModules[0].SourceFile() == nil {
		t.Fatalf("private modules = %#v", privateModules)
	}

	writeFixture(t, filepath.Join(implementation, "private.ts"), `export type DigestView = { value: string };
export const Marker = 1;
`)
	contract.PrivateModules[0].Exports = []string{"DigestView", "Marker"}
	payload, err = json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, contractPath, string(payload)+"\n")
	if _, err := VerifyAll(Config{
		RepositoryRoot: repository,
		Program:        program,
		ContractPaths:  []string{contractPath},
		ScratchRoot:    filepath.Join(root, ".mutation-scratch"),
		Compilation: CompilationDocument{
			Integers: "number", EvaluationOrder: "direct", Concurrency: "disabled",
		},
		TSGoTool: selectedTSGo,
	}); err == nil || !strings.Contains(err.Error(), "is executable or unsupported") {
		t.Fatalf("executable private-module error = %v", err)
	}
}

func TestVerifyRejectsDuplicatePackageOwner(t *testing.T) {
	sourceFile := tsgo.NewFactory().SourceFile(
		nil,
		tsgo.NewFactory().EndOfFile(),
		tsgo.SourceFileData{},
	)
	certificate := &Certificate{byPath: map[string]Implementation{
		"example.test/fast": {
			packagePath: "example.test/fast",
			digest:      "one",
			sourceFile:  sourceFile,
		},
	}}
	if err := certificate.add(Implementation{
		packagePath: "example.test/fast",
		digest:      "two",
		sourceFile:  sourceFile,
	}); err == nil || err.Error() !=
		`certify source implementation admit "example.test/fast": package has multiple implementation owners` {
		t.Fatalf("duplicate implementation error = %v", err)
	}
}

func writeFixture(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
