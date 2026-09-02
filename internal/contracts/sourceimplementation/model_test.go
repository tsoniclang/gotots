package sourceimplementation

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	implementationcontract "github.com/tsoniclang/gotots/internal/contracts/implementation"
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
	sharedCertificationPath := filepath.Join(root, "shared", "core.d.ts")
	writeFixture(t, sharedCertificationPath, `declare module "@fixture/core.js" {
	  export interface CoreValue { readonly value: unknown; }
	}
`)
	writeFixture(t, filepath.Join(implementation, "package.ts"), `import type { CoreValue } from "@fixture/core.js";
type SelectedCore = CoreValue;
export type Digest$Storage = { Hi: bigint; Lo: bigint };
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
	  "files": ["../shared/core.d.ts", "package.ts", "private.ts"]
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
			Integers: "number", EvaluationOrder: "direct",
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
	buildProfile, err := load.NewBuildProfileForToolchain(
		selectedGo.Version(),
		selectedGo.DefaultGOOS(),
		selectedGo.DefaultGOARCH(),
		false,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	certificationSources, err := implementationcontract.LoadCertificationSources(
		[]string{sharedCertificationPath},
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := PrepareAll(Config{
		RepositoryRoot:       repository,
		ContractPaths:        []string{contractPath},
		CertificationSources: certificationSources,
		ScratchRoot:          filepath.Join(root, ".scratch"),
		BuildProfile:         buildProfile,
		Compilation: CompilationDocument{
			Integers: "number", EvaluationOrder: "direct",
		},
		TSGoTool: selectedTSGo,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory:    root,
		Pattern:      ".",
		BuildProfile: buildProfile,
		GoTool:       selectedGo,
	})
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := prepared.Join(program)
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
	assertStagedGeneratedContractLifecycle(
		t,
		root,
		repository,
		selectedTSGo,
		certificate,
		implementationRecord,
	)

	wrongPackage := clonePrepared(prepared)
	wrongPackageRecord := wrongPackage.certificate.byPath["example.test/app/fast"]
	wrongPackageRecord.modulePath = "example.test/other"
	wrongPackage.certificate.byPath[wrongPackageRecord.packagePath] = wrongPackageRecord
	if _, err := wrongPackage.Join(program); err == nil ||
		!strings.Contains(err.Error(), "selected source package identity differs") {
		t.Fatalf("package-identity join error = %v", err)
	}

	wrongBuild := clonePrepared(prepared)
	wrongBuild.buildProfile, err = load.NewBuildProfileForToolchain(
		selectedGo.Version(),
		selectedGo.DefaultGOOS(),
		selectedGo.DefaultGOARCH(),
		false,
		[]string{"joindrift"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrongBuild.Join(program); err == nil ||
		!strings.Contains(err.Error(), "join build profile") {
		t.Fatalf("build-profile join error = %v", err)
	}

	missingPrivateFile := clonePrepared(prepared)
	missingFileRecord := missingPrivateFile.certificate.byPath["example.test/app/fast"]
	missingFileRecord.privateModules[0].goFile = "missing.go"
	missingPrivateFile.certificate.byPath[missingFileRecord.packagePath] = missingFileRecord
	if _, err := missingPrivateFile.Join(program); err == nil ||
		!strings.Contains(err.Error(), "selected Go source file is absent") {
		t.Fatalf("private-file join error = %v", err)
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
	if _, err := PrepareAll(Config{
		RepositoryRoot:       repository,
		ContractPaths:        []string{contractPath},
		CertificationSources: certificationSources,
		ScratchRoot:          filepath.Join(root, ".mutation-scratch"),
		BuildProfile:         buildProfile,
		Compilation: CompilationDocument{
			Integers: "number", EvaluationOrder: "direct",
		},
		TSGoTool: selectedTSGo,
	}); err == nil || !strings.Contains(err.Error(), "is executable or unsupported") {
		t.Fatalf("executable private-module error = %v", err)
	}
}

func assertStagedGeneratedContractLifecycle(
	t *testing.T,
	root string,
	repository string,
	selectedTSGo tsgo.Tool,
	certificate *Certificate,
	implementation Implementation,
) {
	t.Helper()
	const assemblyPath = "fast/package.ts"
	target, err := NewTarget(assemblyPath, implementation.SourceFile())
	if err != nil {
		t.Fatal(err)
	}
	packageTarget, err := NewPackageTarget(implementation.PackagePath(), assemblyPath)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := certificate.PlanGeneratedContracts(
		[]Target{target},
		[]Target{target},
		[]PackageTarget{packageTarget},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Valid() || len(plan.Generated()) != 1 || len(plan.Packages()) != 1 {
		t.Fatal("generated-contract plan is incomplete")
	}
	payload, err := tsgo.EncodeSourceFile(implementation.SourceFile())
	if err != nil {
		t.Fatal(err)
	}
	generatedProtocol := filepath.Join(root, "generated.ast")
	installedProtocol := filepath.Join(root, "installed.ast")
	writeFixture(t, generatedProtocol, string(payload))
	writeFixture(t, installedProtocol, string(payload))
	generated, err := NewStagedTarget(
		assemblyPath,
		generatedProtocol,
		sha256.Sum256(payload),
	)
	if err != nil {
		t.Fatal(err)
	}
	installed, err := NewStagedTarget(
		assemblyPath,
		installedProtocol,
		sha256.Sum256(payload),
	)
	if err != nil {
		t.Fatal(err)
	}
	scratch := filepath.Join(root, "staged-contract-verification")
	if err := VerifyStagedGeneratedContracts(StagedVerificationConfig{
		RepositoryRoot: repository,
		ScratchRoot:    scratch,
		TSGoTool:       selectedTSGo,
		Generated:      []StagedTarget{generated},
		Installed:      []StagedTarget{installed},
		Packages:       plan.Packages(),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(scratch); !os.IsNotExist(err) {
		t.Fatalf("successful verification scratch survived: %v", err)
	}

	mutated := append([]byte(nil), payload...)
	mutated[len(mutated)-1] ^= 1
	writeFixture(t, generatedProtocol, string(mutated))
	mutationScratch := filepath.Join(root, "staged-contract-mutation")
	err = VerifyStagedGeneratedContracts(StagedVerificationConfig{
		RepositoryRoot: repository,
		ScratchRoot:    mutationScratch,
		TSGoTool:       selectedTSGo,
		Generated:      []StagedTarget{generated},
		Installed:      []StagedTarget{installed},
		Packages:       plan.Packages(),
	})
	if err == nil || !strings.Contains(err.Error(), "protocol payload digest changed") {
		t.Fatalf("staged-payload mutation error = %v", err)
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

func clonePrepared(source *Prepared) *Prepared {
	cloned := *source
	cloned.certificate = source.certificate
	cloned.certificate.byPath = make(
		map[string]Implementation,
		len(source.certificate.byPath),
	)
	for path, implementation := range source.certificate.byPath {
		implementation.privateModules = slices.Clone(implementation.privateModules)
		cloned.certificate.byPath[path] = implementation
	}
	return &cloned
}
