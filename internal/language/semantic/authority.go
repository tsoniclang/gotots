package semantic

import (
	"encoding/hex"
	"fmt"
)

type Authority struct {
	kind             AuthorityKind
	toolchainDigest  string
	configuration    string
	packageInput     string
	structureDigest  string
	selectionDigest  string
	artifactDigest   string
	shardDigest      string
	structuralSource string
}

func NewCheckerAuthority(
	toolchainDigest string,
	configurationDigest string,
	packageInputDigest string,
	structureDigest string,
	selectionDigest string,
) (Authority, error) {
	values := []string{
		toolchainDigest,
		configurationDigest,
		packageInputDigest,
		structureDigest,
		selectionDigest,
	}
	for _, value := range values {
		if !fullDigest(value) {
			return Authority{}, fmt.Errorf(
				"checker authority requires full lowercase sha256 digests",
			)
		}
	}
	return Authority{
		kind:            AuthorityChecker,
		toolchainDigest: toolchainDigest,
		configuration:   configurationDigest,
		packageInput:    packageInputDigest,
		structureDigest: structureDigest,
		selectionDigest: selectionDigest,
	}, nil
}

func NewCertifiedProviderAuthority(
	artifactDigest string,
	shardDigest string,
	structuralArtifactDigest string,
) (Authority, error) {
	if !fullDigest(artifactDigest) ||
		!fullDigest(shardDigest) ||
		!fullDigest(structuralArtifactDigest) {
		return Authority{}, fmt.Errorf(
			"provider authority requires full lowercase sha256 digests",
		)
	}
	return Authority{
		kind:             AuthorityCertifiedProvider,
		artifactDigest:   artifactDigest,
		shardDigest:      shardDigest,
		structuralSource: structuralArtifactDigest,
	}, nil
}

func (a Authority) Kind() AuthorityKind      { return a.kind }
func (a Authority) ToolchainDigest() string  { return a.toolchainDigest }
func (a Authority) Configuration() string    { return a.configuration }
func (a Authority) PackageInput() string     { return a.packageInput }
func (a Authority) StructureDigest() string  { return a.structureDigest }
func (a Authority) SelectionDigest() string  { return a.selectionDigest }
func (a Authority) ArtifactDigest() string   { return a.artifactDigest }
func (a Authority) ShardDigest() string      { return a.shardDigest }
func (a Authority) StructuralSource() string { return a.structuralSource }

func (a Authority) Valid() bool {
	switch a.kind {
	case AuthorityChecker:
		return fullDigest(a.toolchainDigest) &&
			fullDigest(a.configuration) &&
			fullDigest(a.packageInput) &&
			fullDigest(a.structureDigest) &&
			fullDigest(a.selectionDigest) &&
			a.artifactDigest == "" &&
			a.shardDigest == "" &&
			a.structuralSource == ""
	case AuthorityCertifiedProvider:
		return a.toolchainDigest == "" &&
			a.configuration == "" &&
			a.packageInput == "" &&
			a.structureDigest == "" &&
			a.selectionDigest == "" &&
			fullDigest(a.artifactDigest) &&
			fullDigest(a.shardDigest) &&
			fullDigest(a.structuralSource)
	default:
		return false
	}
}

func fullDigest(value string) bool {
	raw, err := hex.DecodeString(value)
	return err == nil &&
		len(raw) == 32 &&
		hex.EncodeToString(raw) == value
}
