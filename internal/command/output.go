package command

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/config"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	buildManifestName          = "gotots-manifest.json"
	buildManifestSchemaVersion = 2
)

const projectPackageName = "package.json"

func writePrintPlanTo(
	project config.Project,
	verified verifiedPrintPlan,
	semanticDigest string,
	outputDirectory string,
) (int, error) {
	plan := verified.plan
	if err := plan.validate(outputDirectory); err != nil {
		return 0, err
	}
	client, err := tsgo.StartClientWithTool(project.TSGoTool(), outputDirectory)
	if err != nil {
		return 0, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = client.Close()
		}
	}()
	paths := make([]string, 0, len(plan.files)+3)
	for _, file := range plan.files {
		payload, err := os.ReadFile(file.protocolPath)
		if err != nil {
			return 0, commandError("read staged target AST", err.Error())
		}
		if err := file.verifyProtocolPayload(payload); err != nil {
			return 0, err
		}
		printed, err := client.PrintEncodedSourceFile(payload, tsgo.PrintOptions{})
		if err != nil {
			return 0, err
		}
		if err := writeTargetFile(outputDirectory, file.outputPath, []byte(printed)); err != nil {
			return 0, err
		}
		paths = append(paths, file.outputPath)
	}
	if err := client.Close(); err != nil {
		return 0, err
	}
	closed = true
	if err := plan.removeProtocolScratch(); err != nil {
		return 0, err
	}
	if plan.hasRuntimeManifest {
		if err := writeTargetFile(
			outputDirectory,
			plan.runtimeManifest.outputPath,
			plan.runtimeManifest.payload,
		); err != nil {
			return 0, err
		}
		paths = append(paths, plan.runtimeManifest.outputPath)
	}
	sort.Strings(paths)
	if err := writeTargetFile(outputDirectory, projectPackageName, plan.packageDocument); err != nil {
		return 0, err
	}
	paths = append(paths, projectPackageName)
	paths = append(paths, buildManifestName)
	sort.Strings(paths)
	manifest, err := encodeBuildManifest(
		semanticDigest,
		paths,
		plan.representationTransports,
	)
	if err != nil {
		return 0, err
	}
	if err := writeTargetFile(outputDirectory, buildManifestName, manifest); err != nil {
		return 0, err
	}
	return len(paths), nil
}

func encodeProjectPackage(dependencies []packageDependency) ([]byte, error) {
	document := struct {
		Private      bool              `json:"private"`
		Type         string            `json:"type"`
		Dependencies map[string]string `json:"dependencies"`
	}{
		Private:      true,
		Type:         "module",
		Dependencies: make(map[string]string, len(dependencies)),
	}
	for _, dependency := range dependencies {
		if dependency.name == "" || dependency.version == "" {
			return nil, commandError("encode package", "dependency identity is incomplete")
		}
		if _, duplicate := document.Dependencies[dependency.name]; duplicate {
			return nil, commandError("encode package", "dependency is duplicated")
		}
		document.Dependencies[dependency.name] = dependency.version
	}
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, commandError("encode package", err.Error())
	}
	return append(payload, '\n'), nil
}

func writeTargetFile(root string, relative string, payload []byte) error {
	if filepath.IsAbs(relative) || filepath.ToSlash(filepath.Clean(relative)) != relative ||
		relative == "." || relative == ".." ||
		len(relative) >= 3 && relative[:3] == "../" {
		return commandError("write output", fmt.Sprintf("target path %q is invalid", relative))
	}
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return commandError("write output", err.Error())
	}
	if err := os.WriteFile(target, payload, 0o644); err != nil {
		return commandError("write output", err.Error())
	}
	return nil
}

type representationTransportContractDocument struct {
	SchemaVersion int                       `json:"schemaVersion"`
	Digest        string                    `json:"digest"`
	Callables     []representationTransport `json:"callables"`
}

func encodeBuildManifest(
	semanticDigest string,
	files []string,
	transports []representationTransport,
) ([]byte, error) {
	selected := slices.Clone(files)
	if !slices.IsSorted(selected) {
		return nil, commandError("encode build manifest", "files are not sorted")
	}
	transportContract, err := sealRepresentationTransportContract(transports)
	if err != nil {
		return nil, err
	}
	document := struct {
		SchemaVersion            int                                     `json:"schemaVersion"`
		SemanticDigest           string                                  `json:"semanticDigest"`
		Files                    []string                                `json:"files"`
		RepresentationTransports representationTransportContractDocument `json:"representationTransports"`
	}{
		SchemaVersion:            buildManifestSchemaVersion,
		SemanticDigest:           semanticDigest,
		Files:                    selected,
		RepresentationTransports: transportContract,
	}
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, commandError("encode build manifest", err.Error())
	}
	return append(payload, '\n'), nil
}

func sealRepresentationTransportContract(
	transports []representationTransport,
) (representationTransportContractDocument, error) {
	callables := slices.Clone(transports)
	body := struct {
		SchemaVersion int                       `json:"schemaVersion"`
		Callables     []representationTransport `json:"callables"`
	}{
		SchemaVersion: 1,
		Callables:     callables,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return representationTransportContractDocument{}, commandError(
			"encode representation transport contract",
			err.Error(),
		)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	return representationTransportContractDocument{
		SchemaVersion: body.SchemaVersion,
		Digest:        digest,
		Callables:     callables,
	}, nil
}
