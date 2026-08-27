package callableimplementation

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type StagedTarget struct {
	outputPath   string
	protocolPath string
	protocolHash [sha256.Size]byte
}

func NewStagedTarget(
	outputPath string,
	protocolPath string,
	protocolHash [sha256.Size]byte,
) (StagedTarget, error) {
	if !validTargetPath(outputPath) || !filepath.IsAbs(protocolPath) ||
		protocolHash == ([sha256.Size]byte{}) {
		return StagedTarget{}, &Error{
			Operation: "stage target",
			Subject:   outputPath,
			Reason:    "target evidence is invalid",
		}
	}
	return StagedTarget{
		outputPath: outputPath, protocolPath: protocolPath,
		protocolHash: protocolHash,
	}, nil
}

type StagedModule struct {
	sourcePath           string
	outputPath           string
	sourceDigest         string
	exports              []string
	certificationSources []CertificationSource
}

func NewStagedModule(
	sourcePath string,
	outputPath string,
	sourceDigest string,
	exports []string,
	certificationSources []CertificationSource,
) (StagedModule, error) {
	selectedExports := slices.Clone(exports)
	selectedCertificationSources := slices.Clone(certificationSources)
	digest, digestErr := hex.DecodeString(sourceDigest)
	if !filepath.IsAbs(sourcePath) || filepath.Clean(sourcePath) != sourcePath ||
		!validTargetPath(outputPath) ||
		digestErr != nil || len(digest) != sha256.Size || len(selectedExports) == 0 ||
		!sort.StringsAreSorted(selectedExports) ||
		!sort.SliceIsSorted(selectedCertificationSources, func(left, right int) bool {
			return selectedCertificationSources[left].sourcePath <
				selectedCertificationSources[right].sourcePath
		}) {
		return StagedModule{}, &Error{
			Operation: "stage module",
			Subject:   outputPath,
			Reason:    "module evidence is invalid",
		}
	}
	for index, name := range selectedExports {
		if name == "" || index > 0 && selectedExports[index-1] == name {
			return StagedModule{}, &Error{
				Operation: "stage module",
				Subject:   outputPath,
				Reason:    "module exports are invalid",
			}
		}
	}
	for index, source := range selectedCertificationSources {
		if !source.Valid() || source.sourcePath == sourcePath ||
			index > 0 && selectedCertificationSources[index-1].sourcePath == source.sourcePath {
			return StagedModule{}, &Error{
				Operation: "stage module",
				Subject:   outputPath,
				Reason:    "certification sources are invalid",
			}
		}
	}
	return StagedModule{
		sourcePath: sourcePath, outputPath: outputPath, sourceDigest: sourceDigest,
		exports: selectedExports, certificationSources: selectedCertificationSources,
	}, nil
}

type StagedCallable struct {
	sourceIdentity       string
	sourceSignature      string
	variant              Variant
	implementationOutput string
	implementationExport string
	generated            GeneratedTarget
}

func NewStagedCallable(
	sourceIdentity string,
	sourceSignature string,
	variant Variant,
	implementationOutput string,
	implementationExport string,
	generated GeneratedTarget,
) (StagedCallable, error) {
	if sourceIdentity == "" || sourceSignature == "" || !variant.Valid() ||
		!validTargetPath(implementationOutput) || implementationExport == "" ||
		!generated.Valid() || generated.sourceIdentity != sourceIdentity ||
		generated.variant != variant {
		return StagedCallable{}, &Error{
			Operation: "stage callable",
			Subject:   sourceIdentity,
			Reason:    "callable evidence is invalid",
		}
	}
	return StagedCallable{
		sourceIdentity: sourceIdentity, sourceSignature: sourceSignature,
		variant: variant, implementationOutput: implementationOutput,
		implementationExport: implementationExport, generated: generated,
	}, nil
}

type VerifiedModule struct {
	outputPath string
	sourceFile tsgo.SourceFile
}

func (m VerifiedModule) OutputPath() string          { return m.outputPath }
func (m VerifiedModule) SourceFile() tsgo.SourceFile { return m.sourceFile }
