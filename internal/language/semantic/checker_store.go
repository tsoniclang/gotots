package semantic

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"

	"github.com/tsoniclang/gotots/internal/identity"
)

type CheckerStoreReadStats struct {
	ShardLoads          int
	MaxPackagesResident int
	metrics             Metrics
}

func (stats CheckerStoreReadStats) Metrics() Metrics {
	return stats.metrics.clone()
}

type CheckerPackageContext struct {
	Package            identity.PackageID
	Provenance         PackageProvenance
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

type CheckerStoreWriter struct {
	file      *os.File
	path      string
	manifest  []packageShardManifest
	previous  identity.PackageID
	closed    bool
	metrics   Metrics
	toolchain string
	config    string
}

func NewCheckerStoreWriter() (*CheckerStoreWriter, error) {
	file, err := os.CreateTemp("", "gotots-checker-semantics-*")
	if err != nil {
		return nil, fmt.Errorf("create checker semantic store: %w", err)
	}
	return &CheckerStoreWriter{
		file: file, path: file.Name(),
	}, nil
}

func (writer *CheckerStoreWriter) Append(pkg Package) error {
	if writer == nil || writer.closed || writer.file == nil {
		return fmt.Errorf("checker semantic store writer is closed")
	}
	if !writer.previous.IsZero() &&
		pkg.ID().Compare(writer.previous) <= 0 {
		return fmt.Errorf("checker semantic packages are not canonical")
	}
	authority, err := checkerPackageAuthority(pkg)
	if err != nil {
		return err
	}
	if writer.toolchain == "" {
		writer.toolchain = authority.ToolchainDigest()
		writer.config = authority.Configuration()
	} else if writer.toolchain != authority.ToolchainDigest() ||
		writer.config != authority.Configuration() {
		return fmt.Errorf(
			"checker semantic packages have different checker contexts",
		)
	}
	offset, err := writer.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	hash := sha256.New()
	measurement, err := writeSemanticShard(
		io.MultiWriter(writer.file, hash), pkg,
	)
	if err != nil {
		_ = writer.file.Truncate(offset)
		_, _ = writer.file.Seek(offset, io.SeekStart)
		return err
	}
	var packageMetrics Metrics
	if err := packageMetrics.addMeasuredPackage(
		pkg, measurement,
	); err != nil {
		_ = writer.file.Truncate(offset)
		_, _ = writer.file.Seek(offset, io.SeekStart)
		return err
	}
	entry, err := packageManifestEntry(
		pkg,
		authority,
		offset,
		measurement.encodedBytes,
		fmt.Sprintf("%x", hash.Sum(nil)),
	)
	if err != nil {
		_ = writer.file.Truncate(offset)
		_, _ = writer.file.Seek(offset, io.SeekStart)
		return err
	}
	writer.manifest = append(writer.manifest, entry)
	writer.previous = pkg.ID()
	writer.metrics.merge(packageMetrics)
	return nil
}

func (writer *CheckerStoreWriter) Seal() (*CheckerStore, Metrics, error) {
	if writer == nil || writer.closed || writer.file == nil {
		return nil, Metrics{}, fmt.Errorf(
			"checker semantic store writer is closed",
		)
	}
	if len(writer.manifest) == 0 {
		writer.Abort()
		return nil, Metrics{}, fmt.Errorf(
			"checker semantic store requires at least one package",
		)
	}
	if err := writer.file.Sync(); err != nil {
		writer.Abort()
		return nil, Metrics{}, err
	}
	store := &CheckerStore{
		file: writer.file, path: writer.path,
		toolchain: writer.toolchain, config: writer.config,
		manifest: append(
			[]packageShardManifest(nil), writer.manifest...,
		),
		byPackage: map[identity.PackageID]int{},
		measured:  map[identity.PackageID]bool{},
	}
	for index, entry := range store.manifest {
		packageID, err := identity.ParsePackageID(entry.Package)
		if err != nil {
			writer.Abort()
			return nil, Metrics{}, err
		}
		store.byPackage[packageID] = index
	}
	store.manifestMetrics = writer.metrics.clone()
	writer.file = nil
	writer.path = ""
	writer.closed = true
	return store, writer.metrics.clone(), nil
}

func (writer *CheckerStoreWriter) Abort() {
	if writer == nil || writer.closed {
		return
	}
	writer.closed = true
	if writer.file != nil {
		_ = writer.file.Close()
	}
	if writer.path != "" {
		_ = os.Remove(writer.path)
	}
	writer.file = nil
	writer.path = ""
}

type CheckerStore struct {
	file            *os.File
	path            string
	manifest        []packageShardManifest
	manifestMetrics Metrics
	byPackage       map[identity.PackageID]int
	toolchain       string
	config          string
	projection      sync.Mutex
	mu              sync.Mutex
	stats           CheckerStoreReadStats
	resident        int
	measured        map[identity.PackageID]bool
	closed          bool
}

func (store *CheckerStore) PackageIDs() []identity.PackageID {
	if store == nil {
		return nil
	}
	out := make([]identity.PackageID, 0, len(store.byPackage))
	for packageID := range store.byPackage {
		out = append(out, packageID)
	}
	sort.Slice(out, func(left, right int) bool {
		return out[left].Compare(out[right]) < 0
	})
	return out
}

func (store *CheckerStore) PackageContext(
	packageID identity.PackageID,
) (CheckerPackageContext, bool, error) {
	if store == nil {
		return CheckerPackageContext{}, false, nil
	}
	index, present := store.byPackage[packageID]
	if !present {
		return CheckerPackageContext{}, false, nil
	}
	entry := store.manifest[index]
	definitions, err := parseDefinitions(entry.Definitions)
	if err != nil {
		return CheckerPackageContext{}, false, err
	}
	declarations, err := parseDeclarations(entry.Declarations)
	if err != nil {
		return CheckerPackageContext{}, false, err
	}
	return CheckerPackageContext{
		Package: packageID, Provenance: PackageProvenance(entry.Provenance),
		Definitions: definitions, Declarations: declarations,
		DefinitionCount:    entry.DefinitionCount,
		ResolutionCount:    entry.ResolutionCount,
		DeclarationCount:   entry.DeclarationCount,
		MemberTargetCount:  entry.MemberTargetCount,
		MemberTargetDigest: entry.MemberTargetDigest,
		BindingCount:       entry.BindingCount, TypeCount: entry.TypeCount,
		OperationCount:   entry.OperationCount,
		UnsupportedCount: entry.UnsupportedCount,
		ShardBytes:       entry.ShardBytes, ShardDigest: entry.ShardDigest,
	}, true, nil
}

func (store *CheckerStore) VisitPackage(
	packageID identity.PackageID,
	visit func(Package) error,
) error {
	if store == nil || visit == nil {
		return fmt.Errorf(
			"checker semantic package visit requires store and visitor",
		)
	}
	index, present := store.byPackage[packageID]
	if !present {
		return fmt.Errorf("checker semantic package %s is absent", packageID)
	}
	store.projection.Lock()
	defer store.projection.Unlock()
	if err := store.beginProjection(); err != nil {
		return err
	}
	defer store.endProjection()
	entry := store.manifest[index]
	authority, err := NewCheckerAuthority(
		store.toolchain,
		store.config,
		entry.PackageInput,
		entry.Structure,
		entry.Selection,
	)
	if err != nil {
		return err
	}
	hash := sha256.New()
	shard := io.NewSectionReader(
		store.file, entry.ShardOffset, entry.ShardBytes,
	)
	pkg, err := decodeSemanticShard(
		io.TeeReader(shard, hash), authority, entry,
	)
	if err != nil {
		return fmt.Errorf(
			"checker semantic package %s: %w", packageID, err,
		)
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != entry.ShardDigest {
		return fmt.Errorf("checker semantic shard digest mismatch")
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
	store.recordPackageMetrics(packageID, packageMetrics)
	return visit(pkg)
}

func (store *CheckerStore) beginProjection() error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.closed || store.file == nil {
		return fmt.Errorf("checker semantic store is closed")
	}
	store.stats.ShardLoads++
	store.resident++
	if store.resident > store.stats.MaxPackagesResident {
		store.stats.MaxPackagesResident = store.resident
	}
	return nil
}

func (store *CheckerStore) endProjection() {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.resident--
}

func (store *CheckerStore) recordPackageMetrics(
	packageID identity.PackageID,
	metrics Metrics,
) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.measured[packageID] {
		return
	}
	store.measured[packageID] = true
	store.stats.metrics.merge(metrics)
}

func (store *CheckerStore) ReadStats() CheckerStoreReadStats {
	if store == nil {
		return CheckerStoreReadStats{}
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	stats := store.stats
	stats.metrics = stats.metrics.clone()
	return stats
}

func (store *CheckerStore) ManifestMetrics() Metrics {
	if store == nil {
		return Metrics{}
	}
	return store.manifestMetrics.clone()
}

func (store *CheckerStore) Close() error {
	if store == nil {
		return nil
	}
	store.projection.Lock()
	defer store.projection.Unlock()
	store.mu.Lock()
	if store.closed {
		store.mu.Unlock()
		return nil
	}
	store.closed = true
	file := store.file
	path := store.path
	store.file = nil
	store.path = ""
	store.mu.Unlock()
	var closeErr error
	if file != nil {
		closeErr = file.Close()
	}
	if path != "" {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return closeErr
}

func packageManifestEntry(
	pkg Package,
	authority Authority,
	offset int64,
	encodedBytes int64,
	digest string,
) (packageShardManifest, error) {
	memberTargets, err := pkg.MemberTargetCensus()
	if err != nil {
		return packageShardManifest{}, err
	}
	entry := packageShardManifest{
		Package: pkg.ID().String(), Provenance: uint8(pkg.Provenance()),
		PackageInput: authority.PackageInput(),
		Structure:    authority.StructureDigest(),
		Selection:    authority.SelectionDigest(),
		ShardOffset:  offset, ShardBytes: encodedBytes, ShardDigest: digest,
		DefinitionCount:    pkg.DefinitionCount(),
		ResolutionCount:    pkg.ResolutionCount(),
		DeclarationCount:   pkg.DeclarationCount(),
		MemberTargetCount:  memberTargets.Count(),
		MemberTargetDigest: memberTargets.Digest(),
		BindingCount:       pkg.BindingCount(),
		TypeCount:          pkg.TypeCount(),
		OperationCount:     pkg.OperationCount(),
		UnsupportedCount:   pkg.UnsupportedCount(),
	}
	if err := pkg.VisitDefinitions(func(
		definition DefinitionSemantics,
	) error {
		entry.Definitions = append(
			entry.Definitions, definition.Definition().String(),
		)
		return nil
	}); err != nil {
		return packageShardManifest{}, err
	}
	if err := pkg.VisitDeclarations(func(
		declaration Declaration,
	) error {
		entry.Declarations = append(
			entry.Declarations, declaration.ID().String(),
		)
		return nil
	}); err != nil {
		return packageShardManifest{}, err
	}
	return entry, nil
}
