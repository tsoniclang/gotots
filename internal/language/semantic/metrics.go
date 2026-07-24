package semantic

import (
	"fmt"
	"io"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

const semanticTailLimit = 20

type PackageSize struct {
	Package      identity.PackageID
	EncodedBytes int64
	Records      int
}

type RecordSize struct {
	Package      identity.PackageID
	Identity     string
	EncodedBytes int64
}

type semanticShardMeasurement struct {
	pkg            identity.PackageID
	encodedBytes   int64
	definitionTail []RecordSize
	operationTail  []RecordSize
	typeTail       []RecordSize
}

func newSemanticShardMeasurement(
	pkg identity.PackageID,
) semanticShardMeasurement {
	return semanticShardMeasurement{pkg: pkg}
}

func (measurement *semanticShardMeasurement) observeDefinition(
	record DefinitionSemantics,
	encodedBytes int64,
) {
	measurement.consider(
		&measurement.definitionTail,
		record.Definition().String(),
		encodedBytes,
	)
}

func (measurement *semanticShardMeasurement) observeOperation(
	record Operation,
	encodedBytes int64,
) {
	measurement.consider(
		&measurement.operationTail,
		record.ID().String(),
		encodedBytes,
	)
}

func (measurement *semanticShardMeasurement) observeType(
	record Type,
	encodedBytes int64,
) {
	measurement.consider(
		&measurement.typeTail,
		record.ID().String(),
		encodedBytes,
	)
}

func (measurement *semanticShardMeasurement) consider(
	tail *[]RecordSize,
	identityValue string,
	encodedBytes int64,
) {
	if len(*tail) == semanticTailLimit {
		last := (*tail)[semanticTailLimit-1]
		if encodedBytes < last.EncodedBytes ||
			(encodedBytes == last.EncodedBytes &&
				identityValue >= last.Identity) {
			return
		}
	}
	*tail = append(*tail, RecordSize{
		Package: measurement.pkg, Identity: identityValue,
		EncodedBytes: encodedBytes,
	})
	trimRecordTail(*tail)
	if len(*tail) > semanticTailLimit {
		*tail = (*tail)[:semanticTailLimit]
	}
}

type Metrics struct {
	packages       int
	definitions    int
	resolutions    int
	declarations   int
	memberTargets  int
	bindings       int
	types          int
	operations     int
	unsupported    int
	encodedBytes   int64
	largestRecords int
	packageTail    []PackageSize
	definitionTail []RecordSize
	operationTail  []RecordSize
	typeTail       []RecordSize
}

func MeasurePackages(packages []Package) (Metrics, error) {
	var metrics Metrics
	seen := map[identity.PackageID]bool{}
	for _, pkg := range packages {
		if pkg.ID().IsZero() || seen[pkg.ID()] {
			return Metrics{}, fmt.Errorf(
				"semantic metrics received duplicate package %s",
				pkg.ID(),
			)
		}
		seen[pkg.ID()] = true
		if err := metrics.addPackage(pkg); err != nil {
			return Metrics{}, err
		}
	}
	return metrics, nil
}

func measureShardManifest(
	manifest []packageShardManifest,
) (Metrics, error) {
	var metrics Metrics
	for _, entry := range manifest {
		records := entry.DefinitionCount +
			entry.ResolutionCount +
			entry.DeclarationCount +
			entry.BindingCount +
			entry.TypeCount +
			entry.OperationCount +
			entry.UnsupportedCount
		metrics.packages++
		metrics.definitions += entry.DefinitionCount
		metrics.resolutions += entry.ResolutionCount
		metrics.declarations += entry.DeclarationCount
		metrics.memberTargets += entry.MemberTargetCount
		metrics.bindings += entry.BindingCount
		metrics.types += entry.TypeCount
		metrics.operations += entry.OperationCount
		metrics.unsupported += entry.UnsupportedCount
		metrics.encodedBytes += entry.ShardBytes
		if records > metrics.largestRecords {
			metrics.largestRecords = records
		}
		packageID, err := identity.ParsePackageID(entry.Package)
		if err != nil {
			return Metrics{}, err
		}
		metrics.packageTail = append(
			metrics.packageTail,
			PackageSize{
				Package:      packageID,
				EncodedBytes: entry.ShardBytes,
				Records:      records,
			},
		)
	}
	metrics.trim()
	return metrics, nil
}

func (metrics *Metrics) addPackage(pkg Package) error {
	measurement, err := writeSemanticShard(io.Discard, pkg)
	if err != nil {
		return err
	}
	return metrics.addMeasuredPackage(pkg, measurement)
}

func (metrics *Metrics) addMeasuredPackage(
	pkg Package,
	measurement semanticShardMeasurement,
) error {
	if measurement.pkg != pkg.ID() ||
		measurement.encodedBytes <= 0 {
		return fmt.Errorf(
			"semantic shard measurement disagrees with package %s",
			pkg.ID(),
		)
	}
	records := pkg.DefinitionCount() +
		pkg.ResolutionCount() +
		pkg.DeclarationCount() +
		pkg.BindingCount() +
		pkg.TypeCount() +
		pkg.OperationCount() +
		pkg.UnsupportedCount()
	metrics.packages++
	metrics.definitions += pkg.DefinitionCount()
	metrics.resolutions += pkg.ResolutionCount()
	metrics.declarations += pkg.DeclarationCount()
	memberTargets, err := pkg.MemberTargetCensus()
	if err != nil {
		return err
	}
	metrics.memberTargets += memberTargets.Count()
	metrics.bindings += pkg.BindingCount()
	metrics.types += pkg.TypeCount()
	metrics.operations += pkg.OperationCount()
	metrics.unsupported += pkg.UnsupportedCount()
	metrics.encodedBytes += measurement.encodedBytes
	if records > metrics.largestRecords {
		metrics.largestRecords = records
	}
	metrics.packageTail = append(metrics.packageTail, PackageSize{
		Package:      pkg.ID(),
		EncodedBytes: measurement.encodedBytes,
		Records:      records,
	})
	metrics.definitionTail = append(
		metrics.definitionTail, measurement.definitionTail...,
	)
	metrics.operationTail = append(
		metrics.operationTail, measurement.operationTail...,
	)
	metrics.typeTail = append(
		metrics.typeTail, measurement.typeTail...,
	)
	metrics.trim()
	return nil
}

func (metrics *Metrics) merge(other Metrics) {
	metrics.packages += other.packages
	metrics.definitions += other.definitions
	metrics.resolutions += other.resolutions
	metrics.declarations += other.declarations
	metrics.memberTargets += other.memberTargets
	metrics.bindings += other.bindings
	metrics.types += other.types
	metrics.operations += other.operations
	metrics.unsupported += other.unsupported
	metrics.encodedBytes += other.encodedBytes
	if other.largestRecords > metrics.largestRecords {
		metrics.largestRecords = other.largestRecords
	}
	metrics.packageTail = append(
		metrics.packageTail, other.packageTail...,
	)
	metrics.definitionTail = append(
		metrics.definitionTail, other.definitionTail...,
	)
	metrics.operationTail = append(
		metrics.operationTail, other.operationTail...,
	)
	metrics.typeTail = append(
		metrics.typeTail, other.typeTail...,
	)
	metrics.trim()
}

func (metrics Metrics) clone() Metrics {
	metrics.packageTail = append(
		[]PackageSize(nil), metrics.packageTail...,
	)
	metrics.definitionTail = append(
		[]RecordSize(nil), metrics.definitionTail...,
	)
	metrics.operationTail = append(
		[]RecordSize(nil), metrics.operationTail...,
	)
	metrics.typeTail = append(
		[]RecordSize(nil), metrics.typeTail...,
	)
	return metrics
}

func (metrics *Metrics) trim() {
	sort.Slice(metrics.packageTail, func(left, right int) bool {
		if metrics.packageTail[left].EncodedBytes !=
			metrics.packageTail[right].EncodedBytes {
			return metrics.packageTail[left].EncodedBytes >
				metrics.packageTail[right].EncodedBytes
		}
		return metrics.packageTail[left].Package.Compare(
			metrics.packageTail[right].Package,
		) < 0
	})
	if len(metrics.packageTail) > semanticTailLimit {
		metrics.packageTail = metrics.packageTail[:semanticTailLimit]
	}
	trimRecordTail(metrics.definitionTail)
	trimRecordTail(metrics.operationTail)
	trimRecordTail(metrics.typeTail)
	if len(metrics.definitionTail) > semanticTailLimit {
		metrics.definitionTail =
			metrics.definitionTail[:semanticTailLimit]
	}
	if len(metrics.operationTail) > semanticTailLimit {
		metrics.operationTail =
			metrics.operationTail[:semanticTailLimit]
	}
	if len(metrics.typeTail) > semanticTailLimit {
		metrics.typeTail = metrics.typeTail[:semanticTailLimit]
	}
}

func trimRecordTail(records []RecordSize) {
	sort.Slice(records, func(left, right int) bool {
		if records[left].EncodedBytes != records[right].EncodedBytes {
			return records[left].EncodedBytes >
				records[right].EncodedBytes
		}
		if records[left].Identity != records[right].Identity {
			return records[left].Identity < records[right].Identity
		}
		return records[left].Package.Compare(records[right].Package) < 0
	})
}

func (metrics Metrics) Packages() int      { return metrics.packages }
func (metrics Metrics) Definitions() int   { return metrics.definitions }
func (metrics Metrics) Resolutions() int   { return metrics.resolutions }
func (metrics Metrics) Declarations() int  { return metrics.declarations }
func (metrics Metrics) MemberTargets() int { return metrics.memberTargets }
func (metrics Metrics) Bindings() int      { return metrics.bindings }
func (metrics Metrics) Types() int         { return metrics.types }
func (metrics Metrics) Operations() int    { return metrics.operations }
func (metrics Metrics) Unsupported() int   { return metrics.unsupported }
func (metrics Metrics) EncodedBytes() int64 {
	return metrics.encodedBytes
}
func (metrics Metrics) LargestPackageRecords() int {
	return metrics.largestRecords
}
func (metrics Metrics) LargestPackages() []PackageSize {
	return append([]PackageSize(nil), metrics.packageTail...)
}
func (metrics Metrics) LargestDefinitions() []RecordSize {
	return append([]RecordSize(nil), metrics.definitionTail...)
}
func (metrics Metrics) LargestOperations() []RecordSize {
	return append([]RecordSize(nil), metrics.operationTail...)
}
func (metrics Metrics) LargestTypes() []RecordSize {
	return append([]RecordSize(nil), metrics.typeTail...)
}
