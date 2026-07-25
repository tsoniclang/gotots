package semantic

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"

	"github.com/tsoniclang/gotots/internal/identity"
)

const maxProviderManifestBytes = 64 << 20

type ProviderPackageContext struct {
	Package            identity.PackageID
	Provenance         PackageProvenance
	PackageInput       string
	Structure          string
	Selection          string
	Definitions        []identity.DefinitionID
	Declarations       []identity.SemanticDeclarationID
	DefinitionCount    int
	ResolutionCount    int
	DeclarationCount   int
	MemberTargetCount  int
	MemberTargetDigest string
	BindingCount       int
	TypeCount          int
	OperationCount     int
	UnsupportedCount   int
	ShardBytes         int64
	ShardDigest        string
}

type ProviderReadStats struct {
	ShardLoads                  int
	MaxProviderPackagesResident int
	metrics                     Metrics
}

func (stats ProviderReadStats) Metrics() Metrics {
	return stats.metrics.clone()
}

type ProviderArtifact struct {
	path            string
	digest          string
	context         providerContext
	manifest        []packageShardManifest
	manifestMetrics Metrics
	byPackage       map[identity.PackageID]int
	shardBase       int64
	fileBytes       int64
	projection      sync.Mutex
	mu              sync.Mutex
	stats           ProviderReadStats
	resident        int
	measured        map[identity.PackageID]bool
}

func DecodeProviderArtifact(
	path string,
	expectedDigest string,
) (*ProviderArtifact, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() < providerArtifactHeaderBytes {
		return nil, &artifactError{
			reason: "semantic provider artifact is truncated",
		}
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}
	digest := fmt.Sprintf("%x", hash.Sum(nil))
	if expectedDigest != "" && digest != expectedDigest {
		return nil, &artifactError{
			reason: "semantic provider artifact digest mismatch",
		}
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	header := make([]byte, providerArtifactHeaderBytes)
	if _, err := io.ReadFull(file, header); err != nil {
		return nil, err
	}
	if string(header[:8]) != string(providerArtifactMagic[:]) {
		return nil, &artifactError{
			reason: "semantic provider artifact magic is invalid",
		}
	}
	manifestBytes := binary.BigEndian.Uint64(header[8:])
	if manifestBytes == 0 ||
		manifestBytes > maxProviderManifestBytes ||
		manifestBytes > uint64(info.Size()-providerArtifactHeaderBytes) {
		return nil, &artifactError{
			reason: "semantic provider manifest size is invalid",
		}
	}
	encoded := make([]byte, int(manifestBytes))
	if _, err := io.ReadFull(file, encoded); err != nil {
		return nil, err
	}
	var manifest providerManifest
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf(
			"semantic provider manifest decode failed: %w", err,
		)
	}
	if err := requireJSONEnd(decoder); err != nil {
		return nil, err
	}
	artifact := &ProviderArtifact{
		path:    path,
		digest:  digest,
		context: manifest.Context,
		manifest: append(
			[]packageShardManifest(nil), manifest.Packages...,
		),
		byPackage: map[identity.PackageID]int{},
		measured:  map[identity.PackageID]bool{},
		shardBase: int64(providerArtifactHeaderBytes) +
			int64(manifestBytes),
		fileBytes: info.Size(),
	}
	if err := artifact.validateManifest(); err != nil {
		return nil, err
	}
	artifact.manifestMetrics, err = measureShardManifest(
		artifact.manifest,
	)
	if err != nil {
		return nil, err
	}
	return artifact, nil
}

func (artifact *ProviderArtifact) validateManifest() error {
	context := artifact.context
	if context.Version != ProviderArtifactVersion ||
		!fullDigest(context.ToolchainDigest) ||
		!fullDigest(context.ConfigurationDigest) ||
		context.ContractID == "" ||
		!fullDigest(context.ContractFingerprint) ||
		!fullDigest(context.StructuralArtifactDigest) {
		return &artifactError{
			reason: "semantic provider context is invalid",
		}
	}
	var (
		previous   identity.PackageID
		nextOffset int64
	)
	for index, entry := range artifact.manifest {
		pkg, err := identity.ParsePackageID(entry.Package)
		if err != nil {
			return err
		}
		if (!previous.IsZero() && pkg.Compare(previous) <= 0) ||
			entry.ShardOffset != nextOffset ||
			entry.ShardBytes <= 0 ||
			!fullDigest(entry.ShardDigest) ||
			!fullDigest(entry.PackageInput) ||
			!fullDigest(entry.Structure) ||
			!fullDigest(entry.Selection) ||
			!fullDigest(entry.MemberTargetDigest) ||
			!PackageProvenance(entry.Provenance).Valid() ||
			!providerManifestCountsValid(entry) {
			return &artifactError{
				reason: "semantic provider manifest package is invalid",
			}
		}
		if _, duplicate := artifact.byPackage[pkg]; duplicate {
			return &artifactError{
				reason: "semantic provider manifest repeats package",
			}
		}
		if err := validateManifestCensus(entry); err != nil {
			return err
		}
		artifact.byPackage[pkg] = index
		previous = pkg
		nextOffset += entry.ShardBytes
	}
	if artifact.shardBase+nextOffset != artifact.fileBytes {
		return &artifactError{
			reason: "semantic provider shard extents do not cover artifact",
		}
	}
	return nil
}

func providerManifestCountsValid(entry packageShardManifest) bool {
	if entry.MemberTargetCount < 0 {
		return false
	}
	counts := [...]int{
		entry.DefinitionCount,
		entry.ResolutionCount,
		entry.DeclarationCount,
		entry.BindingCount,
		entry.TypeCount,
		entry.OperationCount,
		entry.UnsupportedCount,
	}
	var total int64
	for _, count := range counts {
		if count < 0 ||
			int64(count) > entry.ShardBytes-total {
			return false
		}
		total += int64(count)
	}
	return true
}

func validateManifestCensus(
	entry packageShardManifest,
) error {
	if entry.DefinitionCount != len(entry.Definitions) ||
		entry.DeclarationCount != len(entry.Declarations) {
		return &artifactError{
			reason: "semantic provider manifest census count mismatch",
		}
	}
	var previousDefinition identity.DefinitionID
	for _, value := range entry.Definitions {
		id, err := identity.ParseDefinitionID(value)
		if err != nil {
			return err
		}
		if !previousDefinition.IsZero() &&
			id.Compare(previousDefinition) <= 0 {
			return &artifactError{
				reason: "semantic provider definitions are not canonical",
			}
		}
		previousDefinition = id
	}
	var previousDeclaration identity.SemanticDeclarationID
	for _, value := range entry.Declarations {
		id, err := identity.ParseSemanticDeclarationID(value)
		if err != nil {
			return err
		}
		if !previousDeclaration.IsZero() &&
			id.Compare(previousDeclaration) <= 0 {
			return &artifactError{
				reason: "semantic provider declarations are not canonical",
			}
		}
		if id.Form() == identity.SemanticDeclarationMember {
			return &artifactError{
				reason: fmt.Sprintf(
					"semantic provider manifest package %s serializes member declaration %s",
					entry.Package, id,
				),
			}
		}
		previousDeclaration = id
	}
	return nil
}

func (artifact *ProviderArtifact) VerifyContext(
	expected ProviderArtifactContext,
	structuralArtifactDigest string,
) error {
	if artifact == nil ||
		artifact.context.ToolchainDigest != expected.ToolchainDigest ||
		artifact.context.ConfigurationDigest !=
			expected.ConfigurationDigest ||
		artifact.context.ContractID != expected.ContractID ||
		artifact.context.ContractFingerprint !=
			expected.ContractFingerprint ||
		artifact.context.StructuralArtifactDigest !=
			structuralArtifactDigest {
		return &artifactError{
			reason: "semantic provider artifact context mismatch",
		}
	}
	return nil
}

func (artifact *ProviderArtifact) Digest() string {
	if artifact == nil {
		return ""
	}
	return artifact.digest
}

func (artifact *ProviderArtifact) StructuralArtifactDigest() string {
	if artifact == nil {
		return ""
	}
	return artifact.context.StructuralArtifactDigest
}

func (artifact *ProviderArtifact) PackageIDs() []identity.PackageID {
	if artifact == nil {
		return nil
	}
	out := make(
		[]identity.PackageID, 0, len(artifact.byPackage),
	)
	for packageID := range artifact.byPackage {
		out = append(out, packageID)
	}
	sort.Slice(out, func(left, right int) bool {
		return out[left].Compare(out[right]) < 0
	})
	return out
}

func (artifact *ProviderArtifact) PackageContext(
	packageID identity.PackageID,
) (ProviderPackageContext, bool, error) {
	if artifact == nil {
		return ProviderPackageContext{}, false, nil
	}
	index, present := artifact.byPackage[packageID]
	if !present {
		return ProviderPackageContext{}, false, nil
	}
	entry := artifact.manifest[index]
	definitions, err := parseDefinitions(entry.Definitions)
	if err != nil {
		return ProviderPackageContext{}, false, err
	}
	declarations, err := parseDeclarations(entry.Declarations)
	if err != nil {
		return ProviderPackageContext{}, false, err
	}
	return ProviderPackageContext{
		Package:            packageID,
		Provenance:         PackageProvenance(entry.Provenance),
		PackageInput:       entry.PackageInput,
		Structure:          entry.Structure,
		Selection:          entry.Selection,
		Definitions:        definitions,
		Declarations:       declarations,
		DefinitionCount:    entry.DefinitionCount,
		ResolutionCount:    entry.ResolutionCount,
		DeclarationCount:   entry.DeclarationCount,
		MemberTargetCount:  entry.MemberTargetCount,
		MemberTargetDigest: entry.MemberTargetDigest,
		BindingCount:       entry.BindingCount,
		TypeCount:          entry.TypeCount,
		OperationCount:     entry.OperationCount,
		UnsupportedCount:   entry.UnsupportedCount,
		ShardBytes:         entry.ShardBytes,
		ShardDigest:        entry.ShardDigest,
	}, true, nil
}

func (artifact *ProviderArtifact) VisitPackage(
	packageID identity.PackageID,
	visit func(Package) error,
) error {
	if artifact == nil {
		return &artifactError{
			reason: "semantic provider artifact is absent",
		}
	}
	if visit == nil {
		return &artifactError{
			reason: "semantic provider package visitor is absent",
		}
	}
	index, present := artifact.byPackage[packageID]
	if !present {
		return &artifactError{
			reason: "semantic provider package is absent",
		}
	}
	artifact.projection.Lock()
	defer artifact.projection.Unlock()
	artifact.beginProjection()
	defer artifact.endProjection()
	entry := artifact.manifest[index]
	file, err := os.Open(artifact.path)
	if err != nil {
		return err
	}
	defer file.Close()
	authority, err := NewCertifiedProviderAuthority(
		artifact.digest,
		entry.ShardDigest,
		artifact.context.StructuralArtifactDigest,
	)
	if err != nil {
		return err
	}
	hash := sha256.New()
	shard := io.NewSectionReader(
		file,
		artifact.shardBase+entry.ShardOffset,
		entry.ShardBytes,
	)
	pkg, err := decodeSemanticShard(
		io.TeeReader(shard, hash),
		authority,
		entry,
	)
	if err != nil {
		return fmt.Errorf(
			"semantic provider package %s: %w",
			entry.Package, err,
		)
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != entry.ShardDigest {
		return &artifactError{
			reason: "semantic provider shard digest mismatch",
		}
	}
	if err := validateProjectedPackage(pkg, entry); err != nil {
		return err
	}
	packageMetrics, err := measureShardManifest(
		[]packageShardManifest{entry},
	)
	if err != nil {
		return err
	}
	artifact.recordPackageMetrics(pkg.ID(), packageMetrics)
	return visit(pkg)
}

func (artifact *ProviderArtifact) beginProjection() {
	artifact.mu.Lock()
	defer artifact.mu.Unlock()
	artifact.stats.ShardLoads++
	artifact.resident++
	if artifact.resident >
		artifact.stats.MaxProviderPackagesResident {
		artifact.stats.MaxProviderPackagesResident =
			artifact.resident
	}
}

func (artifact *ProviderArtifact) endProjection() {
	artifact.mu.Lock()
	defer artifact.mu.Unlock()
	artifact.resident--
}

func (artifact *ProviderArtifact) recordPackageMetrics(
	packageID identity.PackageID,
	metrics Metrics,
) {
	artifact.mu.Lock()
	defer artifact.mu.Unlock()
	if artifact.measured[packageID] {
		return
	}
	artifact.measured[packageID] = true
	artifact.stats.metrics.merge(metrics)
}

func validateProjectedPackage(
	pkg Package,
	entry packageShardManifest,
) error {
	if pkg.ID().String() != entry.Package ||
		uint8(pkg.Provenance()) != entry.Provenance ||
		pkg.DefinitionCount() != entry.DefinitionCount ||
		pkg.ResolutionCount() != entry.ResolutionCount ||
		pkg.DeclarationCount() != entry.DeclarationCount ||
		pkg.BindingCount() != entry.BindingCount ||
		pkg.TypeCount() != entry.TypeCount ||
		pkg.OperationCount() != entry.OperationCount ||
		pkg.UnsupportedCount() != entry.UnsupportedCount {
		return &artifactError{
			reason: "semantic provider shard disagrees with manifest",
		}
	}
	memberTargets, err := pkg.MemberTargetCensus()
	if err != nil {
		return err
	}
	if memberTargets.Count() != entry.MemberTargetCount ||
		memberTargets.Digest() != entry.MemberTargetDigest {
		return &artifactError{
			reason: fmt.Sprintf(
				"semantic provider shard %s member targets disagree with manifest",
				pkg.ID(),
			),
		}
	}
	definitions := make(
		[]string, 0, pkg.DefinitionCount(),
	)
	if err := pkg.VisitDefinitions(func(
		record DefinitionSemantics,
	) error {
		definitions = append(
			definitions, record.Definition().String(),
		)
		return nil
	}); err != nil {
		return err
	}
	declarations := make(
		[]string, 0, pkg.DeclarationCount(),
	)
	if err := pkg.VisitDeclarations(func(
		record Declaration,
	) error {
		declarations = append(
			declarations, record.ID().String(),
		)
		return nil
	}); err != nil {
		return err
	}
	if !equalStrings(definitions, entry.Definitions) ||
		!equalStrings(declarations, entry.Declarations) {
		return &artifactError{
			reason: "semantic provider shard census disagrees with manifest",
		}
	}
	return nil
}

func (artifact *ProviderArtifact) ReadStats() ProviderReadStats {
	if artifact == nil {
		return ProviderReadStats{}
	}
	artifact.mu.Lock()
	defer artifact.mu.Unlock()
	stats := artifact.stats
	stats.metrics = stats.metrics.clone()
	return stats
}

func (artifact *ProviderArtifact) ManifestMetrics() Metrics {
	if artifact == nil {
		return Metrics{}
	}
	return artifact.manifestMetrics.clone()
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
