package command

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestCompileWorkerHandoffRequiresDistinctProcessAndExactPlan(t *testing.T) {
	output := t.TempDir()
	protocol := filepath.Join(output, protocolScratchDirectoryName)
	if err := os.Mkdir(protocol, 0o700); err != nil {
		t.Fatal(err)
	}
	payload := []byte("target AST")
	protocolPath := filepath.Join(protocol, "000000.ast")
	if err := os.WriteFile(protocolPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	generatedProtocol := filepath.Join(
		protocol,
		sourceImplementationProtocolDirectoryName,
	)
	if err := os.Mkdir(generatedProtocol, 0o700); err != nil {
		t.Fatal(err)
	}
	generatedProtocolPath := filepath.Join(generatedProtocol, "000000.ast")
	if err := os.WriteFile(generatedProtocolPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	callableSource := filepath.Join(output, "hot.ts")
	callablePayload := []byte("export function valueFast(): number { return 1; }\n")
	if err := os.WriteFile(callableSource, callablePayload, 0o600); err != nil {
		t.Fatal(err)
	}
	callableDigest := sha256.Sum256(callablePayload)
	plan := printPlan{
		files: []printPlanFile{{
			outputPath:   "program.ts",
			protocolPath: protocolPath,
			protocolHash: sha256.Sum256(payload),
		}},
		protocolDirectory: protocol,
		sourceImplementation: sourceImplementationPrintPlan{
			generated: []printPlanFile{{
				outputPath:   "program.ts",
				protocolPath: generatedProtocolPath,
				protocolHash: sha256.Sum256(payload),
			}},
			packages: []sourceImplementationPackage{{
				packagePath:  "example.test/program",
				assemblyPath: "program.ts",
				exports:      []string{"Value"},
			}},
		},
		hasSourceImplementation: true,
		callableImplementation: callableImplementationPrintPlan{
			modules: []callableImplementationModule{{
				sourcePath:   callableSource,
				outputPath:   "implementations/hot.ts",
				sourceDigest: hex.EncodeToString(callableDigest[:]),
				exports:      []string{"valueFast"},
			}},
			targets: []callableImplementationTarget{{
				sourceIdentity:       "example.test/program|kind=5|receiver=|name=Value",
				sourceSignature:      "func() int|params=|results=",
				variant:              "source",
				implementationOutput: "implementations/hot.ts",
				implementationExport: "valueFast",
				generatedOutput:      "program.ts",
				kind:                 callableImplementationTargetModuleFunction,
				generatedExport:      "Value",
			}},
		},
		hasCallableImplementation: true,
		packageDocument:           []byte("{\"private\":true}\n"),
	}
	digest := "a3a86944267e41877ab54f798340439bd80038e34687c5dedb5dc7dbc857565b"
	handoff, err := encodeCompileWorkerDocument(plan, digest, os.Getpid()+1)
	if err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(output, "handoff.json")
	if err := os.WriteFile(handoffPath, handoff, 0o600); err != nil {
		t.Fatal(err)
	}
	decoded, err := readCompileWorkerDocument(handoffPath, output, os.Getpid()+1)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.semanticDigest != digest || len(decoded.plan.files) != 1 ||
		decoded.plan.files[0].outputPath != "program.ts" ||
		decoded.plan.files[0].protocolHash != plan.files[0].protocolHash ||
		!decoded.plan.hasSourceImplementation ||
		len(decoded.plan.sourceImplementation.generated) != 1 ||
		len(decoded.plan.sourceImplementation.packages) != 1 ||
		!decoded.plan.hasCallableImplementation ||
		len(decoded.plan.callableImplementation.modules) != 1 ||
		len(decoded.plan.callableImplementation.targets) != 1 {
		t.Fatalf("decoded handoff = %#v", decoded)
	}

	sameProcess, err := encodeCompileWorkerDocument(plan, digest, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(handoffPath, sameProcess, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readCompileWorkerDocument(handoffPath, output, os.Getpid()+1); err == nil {
		t.Fatal("same-process compilation handoff was accepted")
	}

	duplicatePlan := plan
	duplicatePlan.files = append(append([]printPlanFile(nil), plan.files...), plan.files[0])
	duplicate, err := encodeCompileWorkerDocument(duplicatePlan, digest, os.Getpid()+1)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(handoffPath, duplicate, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readCompileWorkerDocument(handoffPath, output, os.Getpid()+1); err == nil {
		t.Fatal("duplicate compilation output was accepted")
	}
}

func TestCompileWorkerPathsAreTransactionConfined(t *testing.T) {
	output := filepath.Join(t.TempDir(), "output")
	worker := filepath.Join(output, compileWorkerDirectoryName)
	valid := []string{
		filepath.Join(worker, "project.json"),
		output,
		filepath.Join(worker, "handoff.json"),
	}
	if _, _, _, err := validateCompileWorkerPaths(valid); err != nil {
		t.Fatal(err)
	}
	escaping := append([]string(nil), valid...)
	escaping[2] = filepath.Join(filepath.Dir(output), "handoff.json")
	if _, _, _, err := validateCompileWorkerPaths(escaping); err == nil {
		t.Fatal("escaping compilation handoff was accepted")
	}
}
