package command

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	implementationcontract "github.com/tsoniclang/gotots/internal/contracts/implementation"
	"github.com/tsoniclang/gotots/internal/emit/callableimplementation"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestPreparedCallablePlanExactJoinRejectsHandoffOmission(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(root, "hot.ts"),
		[]byte("export function valueFast(): number { return 1; }\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "runtime-contract.d.ts"),
		[]byte("declare module \"@fixture/runtime.js\" {}\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	sharedCertificationPath := filepath.Join(root, "shared-core.d.ts")
	if err := os.WriteFile(
		sharedCertificationPath,
		[]byte("declare module \"@fixture/core.js\" {}\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	sharedCertificationSources, err := implementationcontract.LoadCertificationSources(
		[]string{sharedCertificationPath},
	)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := load.NewBuildProfileForToolchain(
		"go1.25.0",
		"linux",
		"amd64",
		false,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	claim := callableimplementation.CallableDocument{
		SourceIdentity:   "example.test/program|kind=5|receiver=|name=Value",
		SourceSignature:  "func() int|params=|results=",
		SourceBodyDigest: "a3a86944267e41877ab54f798340439bd80038e34687c5dedb5dc7dbc857565b",
		Variant:          callableimplementation.VariantSource,
		Export:           "valueFast",
	}
	document := callableimplementation.Document{
		SchemaVersion:       callableimplementation.SchemaVersion,
		SourceProgramDigest: strings.Repeat("1", 64),
		Package: callableimplementation.PackageDocument{
			ImportPath: "example.test/program",
			ModulePath: "example.test/program",
		},
		Build: callableimplementation.BuildDocument{
			GoVersion: profile.ToolchainVersion(), GOOS: profile.GOOS(),
			GOARCH: profile.GOARCH(), CGOEnabled: profile.CgoEnabled(),
			BuildTags: profile.Tags(),
		},
		Compilation: callableimplementation.CompilationDocument{
			Integers: "number", EvaluationOrder: "direct",
		},
		Source: "hot.ts", Output: "implementations/hot.ts",
		CertificationSources: []string{"runtime-contract.d.ts"},
		Envelope: implementationcontract.Envelope{
			Kind: implementationcontract.EnvelopeExact,
		},
		Callables: []callableimplementation.CallableDocument{claim},
	}
	payload, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join(root, "contract.json")
	if err := os.WriteFile(contractPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	prepared, err := callableimplementation.PrepareAll(callableimplementation.Config{
		ContractPaths:        []string{contractPath},
		CertificationSources: sharedCertificationSources,
		BuildProfile:         profile,
		Compilation:          document.Compilation,
	})
	if err != nil {
		t.Fatal(err)
	}
	changedDocument := document
	changedDocument.SourceProgramDigest = strings.Repeat("2", 64)
	changedDocument.Output = "implementations/other.ts"
	changedDocument.Callables = append(
		[]callableimplementation.CallableDocument(nil),
		document.Callables...,
	)
	changedDocument.Callables[0].SourceIdentity += ".other"
	changedDocument.Callables[0].Export = "valueOther"
	changedPayload, err := json.Marshal(changedDocument)
	if err != nil {
		t.Fatal(err)
	}
	changedContractPath := filepath.Join(root, "changed-contract.json")
	if err := os.WriteFile(changedContractPath, changedPayload, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := callableimplementation.PrepareAll(callableimplementation.Config{
		ContractPaths:        []string{contractPath, changedContractPath},
		CertificationSources: sharedCertificationSources,
		BuildProfile:         profile,
		Compilation:          document.Compilation,
	}); err == nil || !strings.Contains(err.Error(), "different source snapshots") {
		t.Fatalf("mixed source snapshots error = %v", err)
	}
	module := prepared.Modules()[0]
	certificationSources := module.CertificationSources()
	if len(certificationSources) != 2 {
		t.Fatalf("merged certification sources = %d, want 2", len(certificationSources))
	}
	stagedCertificationSources := make(
		[]callableImplementationCertificationSource,
		len(certificationSources),
	)
	for index, source := range certificationSources {
		stagedCertificationSources[index] = callableImplementationCertificationSource{
			sourcePath: source.SourcePath(), sourceDigest: source.SourceDigest(),
		}
	}
	plan := callableImplementationPrintPlan{
		sourceProgramDigest: document.SourceProgramDigest,
		modules: []callableImplementationModule{{
			sourcePath: module.SourcePath(), outputPath: module.OutputPath(),
			sourceDigest: module.SourceDigest(), exports: []string{claim.Export},
			certificationSources: stagedCertificationSources,
		}},
		targets: []callableImplementationTarget{{
			sourceIdentity: claim.SourceIdentity, sourceSignature: claim.SourceSignature,
			sourceBodyDigest: claim.SourceBodyDigest,
			variant:          string(claim.Variant), implementationOutput: module.OutputPath(),
			implementationExport: claim.Export,
		}},
	}
	if err := exactJoinPreparedCallablePlan(prepared, plan); err != nil {
		t.Fatal(err)
	}

	withoutCertification := plan
	withoutCertification.modules = append([]callableImplementationModule(nil), plan.modules...)
	withoutCertification.modules[0].certificationSources = nil
	if err := exactJoinPreparedCallablePlan(prepared, withoutCertification); err == nil {
		t.Fatal("omitted certification source was accepted")
	}

	withoutCallable := plan
	withoutCallable.targets = nil
	if err := exactJoinPreparedCallablePlan(prepared, withoutCallable); err == nil {
		t.Fatal("omitted callable was accepted")
	}

	changedBodyDigest := plan
	changedBodyDigest.targets = append(
		[]callableImplementationTarget(nil),
		plan.targets...,
	)
	changedBodyDigest.targets[0].sourceBodyDigest = strings.Repeat("0", 64)
	if err := exactJoinPreparedCallablePlan(prepared, changedBodyDigest); err == nil {
		t.Fatal("changed callable source body digest was accepted")
	}

	changedSourceProgram := plan
	changedSourceProgram.sourceProgramDigest = strings.Repeat("2", 64)
	if err := exactJoinPreparedCallablePlan(prepared, changedSourceProgram); err == nil {
		t.Fatal("changed source program digest was accepted")
	}
}
