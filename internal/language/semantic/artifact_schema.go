package semantic

const ProviderArtifactVersion = 6

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
