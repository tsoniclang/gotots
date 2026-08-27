package command

import (
	"encoding/json"
	"os"
	"path/filepath"
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
		SourceIdentity:  "example.test/program|kind=5|receiver=|name=Value",
		SourceSignature: "func() int|params=|results=",
		Variant:         callableimplementation.VariantSource,
		Export:          "valueFast",
	}
	document := callableimplementation.Document{
		SchemaVersion: callableimplementation.SchemaVersion,
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
		ContractPaths: []string{contractPath}, BuildProfile: profile,
		Compilation: document.Compilation,
	})
	if err != nil {
		t.Fatal(err)
	}
	module := prepared.Modules()[0]
	certificationSource := module.CertificationSources()[0]
	plan := callableImplementationPrintPlan{
		modules: []callableImplementationModule{{
			sourcePath: module.SourcePath(), outputPath: module.OutputPath(),
			sourceDigest: module.SourceDigest(), exports: []string{claim.Export},
			certificationSources: []callableImplementationCertificationSource{{
				sourcePath:   certificationSource.SourcePath(),
				sourceDigest: certificationSource.SourceDigest(),
			}},
		}},
		targets: []callableImplementationTarget{{
			sourceIdentity: claim.SourceIdentity, sourceSignature: claim.SourceSignature,
			variant: string(claim.Variant), implementationOutput: module.OutputPath(),
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
}
