package structure

import (
	"sync"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope/contract"
)

// ProviderArtifactVersion is the certified structural-graph schema version.
const ProviderArtifactVersion = 9

// ProviderArtifact is an immutable, request-bound certified structural graph.
// Wire records and lookup indexes are private so validated evidence cannot be
// mutated after admission.
type ProviderArtifact struct {
	version               int
	toolchainFingerprint  string
	catalogFingerprint    string
	buildFlagsFingerprint string
	contractID            string
	contractFingerprint   string
	fileDigests           map[identity.FileID]string
	fileGraphs            map[identity.FileID]FileGraph
	filePackages          map[identity.FileID]identity.PackageID
	packageDigests        map[identity.PackageID]string
	packageGraphs         map[identity.PackageID]PackageGraph
	syntheticPackages     map[identity.PackageID]bool
	factsByID             map[certifiedFactID]CertifiedFact
	manifestFacts         int
	storage               *providerStorage
}

func (a *ProviderArtifact) FileCount() int {
	if a == nil {
		return 0
	}
	return len(a.filePackages)
}

func (a *ProviderArtifact) SyntheticPackageCount() int {
	if a == nil {
		return 0
	}
	return len(a.syntheticPackages)
}

func (a *ProviderArtifact) PackageContextCount() int {
	if a == nil {
		return 0
	}
	return len(a.packageDigests)
}

func (a *ProviderArtifact) FactCount() int {
	if a == nil {
		return 0
	}
	if a.storage != nil {
		return a.storage.factCount
	}
	if a.manifestFacts != 0 {
		return a.manifestFacts
	}
	return len(a.factsByID)
}

// CertifiedFact is one immutable provider-produced selection fact.
type CertifiedFact struct {
	definition     identity.DefinitionID
	kind           contract.SelectionFactKind
	value          bool
	producerDigest string
	evidenceDigest string
}

func NewCertifiedFact(
	definition identity.DefinitionID,
	kind contract.SelectionFactKind,
	value bool,
	producerDigest string,
	evidenceDigest string,
) (CertifiedFact, error) {
	if definition.IsZero() ||
		!kind.Valid() ||
		!validSHA256(producerDigest) ||
		!validSHA256(evidenceDigest) {
		return CertifiedFact{}, providerArtifactError(
			"selection fact has invalid identity or digest",
		)
	}
	return CertifiedFact{
		definition:     definition,
		kind:           kind,
		value:          value,
		producerDigest: producerDigest,
		evidenceDigest: evidenceDigest,
	}, nil
}

func (f CertifiedFact) Definition() identity.DefinitionID {
	return f.definition
}
func (f CertifiedFact) Kind() contract.SelectionFactKind { return f.kind }
func (f CertifiedFact) Value() bool                      { return f.value }
func (f CertifiedFact) ProducerDigest() string           { return f.producerDigest }
func (f CertifiedFact) EvidenceDigest() string           { return f.evidenceDigest }

type certifiedFactID struct {
	definition identity.DefinitionID
	kind       contract.SelectionFactKind
}

// providerContextRecord is the first record of the canonical gzip-compressed
// JSON-lines transport. File/package/fact records follow in canonical order so
// encoding and admission are bounded to one record at a time.
type providerContextRecord struct {
	Version               int    `json:"version"`
	ToolchainFingerprint  string `json:"toolchainFingerprint"`
	CatalogFingerprint    string `json:"catalogFingerprint"`
	BuildFlagsFingerprint string `json:"buildFlagsFingerprint"`
	ContractID            string `json:"contractId"`
	ContractFingerprint   string `json:"contractFingerprint"`
}

type providerStreamRecord struct {
	Record  string                 `json:"record"`
	Context *providerContextRecord `json:"context,omitempty"`
	File    *artifactFile          `json:"file,omitempty"`
	Package *artifactPackage       `json:"package,omitempty"`
	Fact    *artifactFact          `json:"selectionFact,omitempty"`
	Seal    string                 `json:"seal,omitempty"`
}

type providerManifest struct {
	Version  int                       `json:"version"`
	Context  providerContextRecord     `json:"context"`
	Packages []providerManifestPackage `json:"packages"`
}

type providerManifestPackage struct {
	Package     string   `json:"package"`
	InputDigest string   `json:"inputDigest"`
	Files       []string `json:"files"`
	Synthetic   bool     `json:"synthetic"`
	FactCount   int      `json:"factCount"`
	ShardBytes  int64    `json:"shardBytes"`
	ShardDigest string   `json:"shardDigest"`
}

type providerShard struct {
	offset      int64
	bytes       int64
	digest      string
	factCount   int
	synthetic   bool
	files       []identity.FileID
	inputDigest string
}

type providerStorage struct {
	mu        sync.Mutex
	path      string
	shards    map[identity.PackageID]providerShard
	loaded    map[identity.PackageID]*ProviderArtifact
	factCount int
}

type artifactFact struct {
	Definition     string `json:"definition"`
	Kind           uint8  `json:"kind"`
	Value          bool   `json:"value"`
	ProducerDigest string `json:"producerDigest"`
	EvidenceDigest string `json:"evidenceDigest"`
}

type artifactPackage struct {
	Package     string               `json:"package"`
	InputDigest string               `json:"inputDigest"`
	Files       []string             `json:"files"`
	Owners      []string             `json:"syntheticOwners,omitempty"`
	Definitions []artifactDefinition `json:"definitions,omitempty"`
	Sites       []artifactSite       `json:"sites,omitempty"`
	Headers     []artifactHeader     `json:"headers,omitempty"`
	Boundaries  []artifactBoundary   `json:"boundaries,omitempty"`
}

type artifactFile struct {
	File        string               `json:"file"`
	ByteDigest  string               `json:"byteDigest"`
	Occurrences []artifactOccurrence `json:"occurrences"`
	Owner       artifactOwner        `json:"owner"`
	Anchors     []string             `json:"anchors"`
	Definitions []artifactDefinition `json:"definitions"`
	Sites       []artifactSite       `json:"sites"`
	Headers     []artifactHeader     `json:"headers"`
	Boundaries  []artifactBoundary   `json:"boundaries"`
	Mappings    []artifactCheckedMap `json:"checkedMappings"`
}

type artifactOwner struct {
	Members    []string            `json:"members"`
	Directives []artifactDirective `json:"directives"`
}

type artifactOccurrence struct {
	ID      string      `json:"id"`
	Kind    uint16      `json:"kind"`
	Parent  string      `json:"parent,omitempty"`
	Edge    uint16      `json:"edge"`
	Ordinal int         `json:"ordinal"`
	Span    Span        `json:"span"`
	Display DisplaySpan `json:"display"`
	Token   uint16      `json:"token"`
}

type artifactDirective struct {
	Kind    uint16      `json:"kind"`
	Tool    string      `json:"tool"`
	Name    string      `json:"name"`
	Args    string      `json:"args"`
	Span    Span        `json:"span"`
	Display DisplaySpan `json:"display"`
}

type artifactDefinition struct {
	ID       string `json:"id"`
	Owner    string `json:"owner"`
	Header   string `json:"header"`
	Boundary string `json:"boundary"`
	Name     string `json:"name"`
}

type artifactSite struct {
	Kind             uint8  `json:"kind"`
	Definition       string `json:"definition"`
	Owner            string `json:"owner"`
	ParentDefinition string `json:"parentDefinition,omitempty"`
	Terminal         string `json:"terminal,omitempty"`
}

type artifactHeader struct {
	ID      string   `json:"id"`
	Digest  string   `json:"digest"`
	Members []string `json:"members"`
}

type artifactBoundary struct {
	ID             string          `json:"id"`
	Kind           uint8           `json:"kind"`
	Entries        []artifactEntry `json:"entries"`
	CombinedDigest string          `json:"combinedDigest"`
	Implicit       uint8           `json:"implicit"`
	Synthetic      uint8           `json:"synthetic"`
}

type artifactEntry struct {
	ID   string `json:"id"`
	Hash string `json:"hash"`
}

type artifactCheckedMap struct {
	Definition    string `json:"definition"`
	OriginLine    int    `json:"originLine"`
	OriginColumn  int    `json:"originColumn"`
	OriginMatch   uint8  `json:"originMatch"`
	CheckedDigest string `json:"checkedDigest"`
}
