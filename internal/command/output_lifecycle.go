package command

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const protocolScratchDirectoryName = ".gotots-protocol"

type printPlan struct {
	files              []printPlanFile
	protocolDirectory  string
	runtimeManifest    printPlanArtifact
	hasRuntimeManifest bool
	packageDocument    []byte
}

type printPlanFile struct {
	outputPath   string
	protocolPath string
	protocolHash [sha256.Size]byte
}

type printPlanArtifact struct {
	outputPath string
	payload    []byte
}

type packageDependency struct {
	name    string
	version string
}

func stagePrintPlan(
	outputDirectory string,
	emission emit.ProgramEmission,
) (printPlan, error) {
	protocolDirectory := filepath.Join(
		outputDirectory,
		protocolScratchDirectoryName,
	)
	if err := os.Mkdir(protocolDirectory, 0o700); err != nil {
		return printPlan{}, commandError("create protocol staging", err.Error())
	}
	targetFiles := emission.Files()
	files := make([]printPlanFile, 0, len(targetFiles))
	for index, targetFile := range targetFiles {
		payload, err := tsgo.EncodeSourceFile(targetFile.SourceFile())
		if err != nil {
			return printPlan{}, commandError(
				"encode target AST",
				fmt.Sprintf("%s: %v", targetFile.OutputPath(), err),
			)
		}
		protocolPath := filepath.Join(
			protocolDirectory,
			fmt.Sprintf("%06d.ast", index),
		)
		if err := os.WriteFile(protocolPath, payload, 0o600); err != nil {
			return printPlan{}, commandError("stage target AST", err.Error())
		}
		files = append(files, printPlanFile{
			outputPath:   targetFile.OutputPath(),
			protocolPath: protocolPath,
			protocolHash: sha256.Sum256(payload),
		})
	}
	dependencies := emission.PackageDependencies()
	projectDependencies := make([]packageDependency, len(dependencies))
	for index, dependency := range dependencies {
		projectDependencies[index] = packageDependency{
			name:    dependency.Name(),
			version: dependency.Version(),
		}
	}
	packageDocument, err := encodeProjectPackage(projectDependencies)
	if err != nil {
		return printPlan{}, err
	}
	plan := printPlan{
		files:             files,
		protocolDirectory: protocolDirectory,
		packageDocument:   packageDocument,
	}
	if runtimePackage, ok := emission.RuntimePackage(); ok {
		plan.runtimeManifest = printPlanArtifact{
			outputPath: runtimePackage.ManifestPath(),
			payload:    runtimePackage.Manifest(),
		}
		plan.hasRuntimeManifest = true
	}
	return plan, nil
}

func (f printPlanFile) verifyProtocolPayload(payload []byte) error {
	if sha256.Sum256(payload) != f.protocolHash {
		return commandError("verify staged target AST", "protocol payload digest changed")
	}
	return nil
}

func (p printPlan) validate(outputDirectory string) error {
	expectedProtocolDirectory := filepath.Join(
		outputDirectory,
		protocolScratchDirectoryName,
	)
	if p.protocolDirectory != expectedProtocolDirectory {
		return commandError("validate print plan", "protocol directory is not output-owned")
	}
	if len(p.files) == 0 {
		return commandError("validate print plan", "target file set is empty")
	}
	for index, file := range p.files {
		expected := filepath.Join(
			p.protocolDirectory,
			fmt.Sprintf("%06d.ast", index),
		)
		if file.outputPath == "" || file.protocolPath != expected {
			return commandError("validate print plan", "target file identity is invalid")
		}
	}
	if len(p.packageDocument) == 0 {
		return commandError("validate print plan", "project package is absent")
	}
	if p.hasRuntimeManifest &&
		(p.runtimeManifest.outputPath == "" || len(p.runtimeManifest.payload) == 0) {
		return commandError("validate print plan", "runtime manifest is incomplete")
	}
	return nil
}

func (p printPlan) removeProtocolScratch() error {
	for _, file := range p.files {
		if err := os.Remove(file.protocolPath); err != nil {
			return commandError("remove staged target AST", err.Error())
		}
	}
	if err := os.Remove(p.protocolDirectory); err != nil {
		return commandError("remove protocol staging", err.Error())
	}
	return nil
}
