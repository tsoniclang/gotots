package command

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	protocolScratchDirectoryName                  = ".gotots-protocol"
	sourceImplementationProtocolDirectoryName     = "source-implementation-generated"
	sourceImplementationVerificationDirectoryName = ".gotots-source-implementation-verify"
)

type printPlan struct {
	files                   []printPlanFile
	protocolDirectory       string
	sourceImplementation    sourceImplementationPrintPlan
	hasSourceImplementation bool
	runtimeManifest         printPlanArtifact
	hasRuntimeManifest      bool
	packageDocument         []byte
}

type compiledPrintPlan struct {
	plan printPlan
}

type verifiedPrintPlan struct {
	plan printPlan
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

type sourceImplementationPrintPlan struct {
	generated []printPlanFile
	packages  []sourceImplementationPackage
}

type sourceImplementationPackage struct {
	packagePath  string
	assemblyPath string
	exports      []string
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
	if verification, ok := emission.SourceImplementationPlan(); ok {
		generatedDirectory := filepath.Join(
			protocolDirectory,
			sourceImplementationProtocolDirectoryName,
		)
		if err := os.Mkdir(generatedDirectory, 0o700); err != nil {
			return printPlan{}, commandError(
				"create source-implementation protocol staging",
				err.Error(),
			)
		}
		generatedTargets := verification.Generated()
		sort.Slice(generatedTargets, func(left, right int) bool {
			return generatedTargets[left].OutputPath() < generatedTargets[right].OutputPath()
		})
		generated := make([]printPlanFile, len(generatedTargets))
		for index, target := range generatedTargets {
			payload, err := tsgo.EncodeSourceFile(target.SourceFile())
			if err != nil {
				return printPlan{}, commandError(
					"encode source-implementation contract AST",
					fmt.Sprintf("%s: %v", target.OutputPath(), err),
				)
			}
			protocolPath := filepath.Join(
				generatedDirectory,
				fmt.Sprintf("%06d.ast", index),
			)
			if err := os.WriteFile(protocolPath, payload, 0o600); err != nil {
				return printPlan{}, commandError(
					"stage source-implementation contract AST",
					err.Error(),
				)
			}
			generated[index] = printPlanFile{
				outputPath:   target.OutputPath(),
				protocolPath: protocolPath,
				protocolHash: sha256.Sum256(payload),
			}
		}
		contractPackages := verification.Packages()
		packages := make([]sourceImplementationPackage, len(contractPackages))
		for index, selected := range contractPackages {
			packages[index] = sourceImplementationPackage{
				packagePath:  selected.PackagePath(),
				assemblyPath: selected.AssemblyPath(),
				exports:      selected.Exports(),
			}
		}
		plan.sourceImplementation = sourceImplementationPrintPlan{
			generated: generated,
			packages:  packages,
		}
		plan.hasSourceImplementation = true
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
	outputPaths := make(map[string]struct{}, len(p.files))
	for index, file := range p.files {
		expected := filepath.Join(
			p.protocolDirectory,
			fmt.Sprintf("%06d.ast", index),
		)
		if file.outputPath == "" || file.protocolPath != expected {
			return commandError("validate print plan", "target file identity is invalid")
		}
		if _, duplicate := outputPaths[file.outputPath]; duplicate {
			return commandError("validate print plan", "target file identity is duplicated")
		}
		outputPaths[file.outputPath] = struct{}{}
	}
	if p.hasSourceImplementation {
		generatedDirectory := filepath.Join(
			p.protocolDirectory,
			sourceImplementationProtocolDirectoryName,
		)
		if len(p.sourceImplementation.generated) == 0 ||
			len(p.sourceImplementation.packages) == 0 {
			return commandError(
				"validate print plan",
				"source-implementation verification is incomplete",
			)
		}
		generatedPaths := make(map[string]struct{}, len(p.sourceImplementation.generated))
		for index, file := range p.sourceImplementation.generated {
			expected := filepath.Join(
				generatedDirectory,
				fmt.Sprintf("%06d.ast", index),
			)
			if file.outputPath == "" || file.protocolPath != expected {
				return commandError(
					"validate print plan",
					"source-implementation target identity is invalid",
				)
			}
			if _, duplicate := generatedPaths[file.outputPath]; duplicate {
				return commandError(
					"validate print plan",
					"source-implementation target identity is duplicated",
				)
			}
			generatedPaths[file.outputPath] = struct{}{}
		}
		packagePaths := make(map[string]struct{}, len(p.sourceImplementation.packages))
		for _, selected := range p.sourceImplementation.packages {
			if selected.packagePath == "" || selected.assemblyPath == "" ||
				len(selected.exports) == 0 {
				return commandError(
					"validate print plan",
					"source-implementation package contract is invalid",
				)
			}
			if _, duplicate := packagePaths[selected.packagePath]; duplicate {
				return commandError(
					"validate print plan",
					"source-implementation package contract is duplicated",
				)
			}
			if _, ok := outputPaths[selected.assemblyPath]; !ok {
				return commandError(
					"validate print plan",
					"installed source-implementation assembly is absent",
				)
			}
			if _, ok := generatedPaths[selected.assemblyPath]; !ok {
				return commandError(
					"validate print plan",
					"ordinary source-implementation assembly is absent",
				)
			}
			packagePaths[selected.packagePath] = struct{}{}
		}
	} else if len(p.sourceImplementation.generated) != 0 ||
		len(p.sourceImplementation.packages) != 0 {
		return commandError(
			"validate print plan",
			"unselected source-implementation verification survived",
		)
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
	if p.hasSourceImplementation {
		for _, file := range p.sourceImplementation.generated {
			if err := os.Remove(file.protocolPath); err != nil {
				return commandError(
					"remove source-implementation staged target AST",
					err.Error(),
				)
			}
		}
		if err := os.Remove(filepath.Join(
			p.protocolDirectory,
			sourceImplementationProtocolDirectoryName,
		)); err != nil {
			return commandError(
				"remove source-implementation protocol staging",
				err.Error(),
			)
		}
	}
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
