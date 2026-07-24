package semantic

import "fmt"

const ProviderArtifactVersion = 5

type providerContext struct {
	Version                  int    `json:"version"`
	ToolchainDigest          string `json:"toolchainDigest"`
	ConfigurationDigest      string `json:"configurationDigest"`
	ContractID               string `json:"contractId"`
	ContractFingerprint      string `json:"contractFingerprint"`
	StructuralArtifactDigest string `json:"structuralArtifactDigest"`
}

type providerManifest struct {
	Context  providerContext        `json:"context"`
	Packages []packageShardManifest `json:"packages"`
}

type packageShardManifest struct {
	Package            string   `json:"package"`
	Provenance         uint8    `json:"provenance"`
	PackageInput       string   `json:"packageInputDigest"`
	Structure          string   `json:"structureDigest"`
	Selection          string   `json:"selectionDigest"`
	Definitions        []string `json:"definitions"`
	Declarations       []string `json:"declarations"`
	DefinitionCount    int      `json:"definitionCount"`
	ResolutionCount    int      `json:"resolutionCount"`
	DeclarationCount   int      `json:"declarationCount"`
	MemberTargetCount  int      `json:"memberTargetCount"`
	MemberTargetDigest string   `json:"memberTargetDigest"`
	BindingCount       int      `json:"bindingCount"`
	TypeCount          int      `json:"typeCount"`
	OperationCount     int      `json:"operationCount"`
	UnsupportedCount   int      `json:"unsupportedCount"`
	ShardOffset        int64    `json:"shardOffset"`
	ShardBytes         int64    `json:"shardBytes"`
	ShardDigest        string   `json:"shardDigest"`
}

type wireModuleReference uint64
type wireOwnerReference uint64
type wirePackageReference uint64
type wireFileReference uint64
type wireSpanReference uint64
type wireOccurrenceReference uint64
type wireDefinitionReference uint64
type wireTypeReference uint64
type wireDeclarationReference uint64
type wireBindingReference uint64
type wireOperationReference uint64
type wireUnsupportedReference uint64

func newWireReference[Reference ~uint64](
	index uint64,
) (Reference, error) {
	if index == 0 {
		return 0, fmt.Errorf("semantic wire reference is zero")
	}
	return Reference(index), nil
}

type wireReferenceRange[Reference ~uint64] struct {
	Start  uint64      `json:"start"`
	Count  uint64      `json:"count"`
	Values []Reference `json:"values,omitempty"`
}

type wireIntegerRange struct {
	Start  uint64 `json:"start"`
	Count  uint64 `json:"count"`
	Values []int  `json:"values,omitempty"`
}

type semanticShardCounts struct {
	Modules      uint64 `json:"modules"`
	Owners       uint64 `json:"owners"`
	Packages     uint64 `json:"packages"`
	Files        uint64 `json:"files"`
	Spans        uint64 `json:"spans"`
	Occurrences  uint64 `json:"occurrences"`
	Definitions  uint64 `json:"definitions"`
	Types        uint64 `json:"types"`
	Declarations uint64 `json:"declarations"`
	Bindings     uint64 `json:"bindings"`
	Operations   uint64 `json:"operations"`
	Unsupported  uint64 `json:"unsupported"`

	DefinitionRecords  uint64 `json:"definitionRecords"`
	ResolutionRecords  uint64 `json:"resolutionRecords"`
	DeclarationRecords uint64 `json:"declarationRecords"`
	BindingRecords     uint64 `json:"bindingRecords"`
	TypeRecords        uint64 `json:"typeRecords"`
	OperationRecords   uint64 `json:"operationRecords"`
	UnsupportedRecords uint64 `json:"unsupportedRecords"`
}

func (counts semanticShardCounts) total() (uint64, bool) {
	values := [...]uint64{
		counts.Modules,
		counts.Owners,
		counts.Packages,
		counts.Files,
		counts.Spans,
		counts.Occurrences,
		counts.Definitions,
		counts.Types,
		counts.Declarations,
		counts.Bindings,
		counts.Operations,
		counts.Unsupported,
		counts.DefinitionRecords,
		counts.ResolutionRecords,
		counts.DeclarationRecords,
		counts.BindingRecords,
		counts.TypeRecords,
		counts.OperationRecords,
		counts.UnsupportedRecords,
	}
	var total uint64
	for _, value := range values {
		if value > ^uint64(0)-total {
			return 0, false
		}
		total += value
	}
	return total, true
}

func (counts semanticShardCounts) validate(
	entry packageShardManifest,
) error {
	total, valid := counts.total()
	if !valid || entry.ShardBytes <= 0 ||
		total > uint64(entry.ShardBytes) ||
		counts.DefinitionRecords != uint64(entry.DefinitionCount) ||
		counts.ResolutionRecords != uint64(entry.ResolutionCount) ||
		counts.DeclarationRecords != uint64(entry.DeclarationCount) ||
		counts.BindingRecords != uint64(entry.BindingCount) ||
		counts.TypeRecords != uint64(entry.TypeCount) ||
		counts.OperationRecords != uint64(entry.OperationCount) ||
		counts.UnsupportedRecords != uint64(entry.UnsupportedCount) {
		return fmt.Errorf(
			"semantic normalized shard counts disagree with manifest",
		)
	}
	return nil
}
