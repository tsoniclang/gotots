package semantic

import (
	"encoding/json"
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

type Metrics struct {
	packages       int
	definitions    int
	resolutions    int
	declarations   int
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

func measureProviderManifest(
	manifest []providerManifestPackage,
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
	encodedBytes, err := writeProviderShard(io.Discard, pkg)
	if err != nil {
		return err
	}
	return metrics.addMeasuredPackage(pkg, encodedBytes)
}

func (metrics *Metrics) addMeasuredPackage(
	pkg Package,
	encodedBytes int64,
) error {
	records := len(pkg.definitions) +
		len(pkg.resolutions) +
		len(pkg.declarations) +
		len(pkg.bindings) +
		len(pkg.types) +
		len(pkg.operations) +
		len(pkg.unsupported)
	metrics.packages++
	metrics.definitions += len(pkg.definitions)
	metrics.resolutions += len(pkg.resolutions)
	metrics.declarations += len(pkg.declarations)
	metrics.bindings += len(pkg.bindings)
	metrics.types += len(pkg.types)
	metrics.operations += len(pkg.operations)
	metrics.unsupported += len(pkg.unsupported)
	metrics.encodedBytes += encodedBytes
	if records > metrics.largestRecords {
		metrics.largestRecords = records
	}
	metrics.packageTail = append(metrics.packageTail, PackageSize{
		Package:      pkg.ID(),
		EncodedBytes: encodedBytes,
		Records:      records,
	})
	for _, record := range pkg.definitions {
		size, err := encodedRecordBytes(encodeDefinition(record))
		if err != nil {
			return err
		}
		metrics.definitionTail = append(
			metrics.definitionTail,
			RecordSize{
				Package:      pkg.ID(),
				Identity:     record.Definition().String(),
				EncodedBytes: size,
			},
		)
	}
	for _, record := range pkg.operations {
		size, err := encodedRecordBytes(encodeOperation(record))
		if err != nil {
			return err
		}
		metrics.operationTail = append(
			metrics.operationTail,
			RecordSize{
				Package: pkg.ID(), Identity: record.ID().String(),
				EncodedBytes: size,
			},
		)
	}
	for _, record := range pkg.types {
		size, err := encodedRecordBytes(encodeType(record))
		if err != nil {
			return err
		}
		metrics.typeTail = append(
			metrics.typeTail,
			RecordSize{
				Package: pkg.ID(), Identity: record.ID().String(),
				EncodedBytes: size,
			},
		)
	}
	metrics.trim()
	return nil
}

func encodedRecordBytes(record any) (int64, error) {
	encoded, err := json.Marshal(record)
	if err != nil {
		return 0, fmt.Errorf(
			"encode semantic metric record: %w", err,
		)
	}
	return int64(len(encoded)), nil
}

func (metrics *Metrics) merge(other Metrics) {
	metrics.packages += other.packages
	metrics.definitions += other.definitions
	metrics.resolutions += other.resolutions
	metrics.declarations += other.declarations
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
		return metrics.packageTail[left].Package.String() <
			metrics.packageTail[right].Package.String()
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
		return records[left].Package.String() <
			records[right].Package.String()
	})
}

func (metrics Metrics) Packages() int     { return metrics.packages }
func (metrics Metrics) Definitions() int  { return metrics.definitions }
func (metrics Metrics) Resolutions() int  { return metrics.resolutions }
func (metrics Metrics) Declarations() int { return metrics.declarations }
func (metrics Metrics) Bindings() int     { return metrics.bindings }
func (metrics Metrics) Types() int        { return metrics.types }
func (metrics Metrics) Operations() int   { return metrics.operations }
func (metrics Metrics) Unsupported() int  { return metrics.unsupported }
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
