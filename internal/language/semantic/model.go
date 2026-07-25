package semantic

import (
	"fmt"
	"sync"

	"github.com/tsoniclang/gotots/internal/identity"
)

type PackageInput struct {
	ID            identity.PackageID
	Provenance    PackageProvenance
	Definitions   []DefinitionSemantics
	Resolutions   []OccurrenceResolution
	Declarations  []Declaration
	Bindings      []Binding
	Types         []Type
	TypeWitnesses []TypeWitness
	Operations    []Operation
	Unsupported   []Unsupported
}

type Package struct {
	id            identity.PackageID
	provenance    PackageProvenance
	identities    packageIdentityTable
	authorities   packageAuthorityTable
	definitions   packageDefinitionStore
	resolutions   packageResolutionStore
	declarations  packageDeclarationStore
	bindings      packageBindingStore
	types         packageTypeStore
	witnesses     packageTypeWitnessStore
	operations    packageOperationStore
	operationView *packageOperationProjection
	unsupported   packageUnsupportedStore
	memberTargets MemberTargetCensus
}

func NewPackage(input PackageInput) (Package, error) {
	return newPackage(input)
}

func newPackage(input PackageInput) (Package, error) {
	if input.ID.IsZero() {
		return Package{}, fmt.Errorf(
			"semantic package requires package identity",
		)
	}
	if !input.Provenance.Valid() {
		return Package{}, fmt.Errorf(
			"semantic package requires closed provenance",
		)
	}
	var normalized normalizedPackageBuilder
	for _, record := range input.Definitions {
		normalized.addDefinition(record)
	}
	for _, record := range input.Resolutions {
		normalized.addResolution(record)
	}
	for _, record := range input.Declarations {
		normalized.addDeclaration(record)
	}
	for _, record := range input.Bindings {
		normalized.addBinding(record)
	}
	for _, record := range input.Types {
		normalized.addType(record)
	}
	for _, record := range input.TypeWitnesses {
		normalized.addTypeWitness(record)
	}
	for _, record := range input.Operations {
		normalized.addOperation(record)
	}
	for _, record := range input.Unsupported {
		normalized.addUnsupported(record)
	}
	return newPackageFromBuilder(
		input.ID, input.Provenance, &normalized,
	)
}

func newPackageFromBuilder(
	id identity.PackageID,
	provenance PackageProvenance,
	normalized *normalizedPackageBuilder,
) (Package, error) {
	stores, err := normalized.seal()
	if err != nil {
		return Package{}, err
	}
	return newPackageFromStores(id, provenance, stores)
}

func newPackageFromStores(
	id identity.PackageID,
	provenance PackageProvenance,
	stores normalizedPackageStores,
) (Package, error) {
	if id.IsZero() || !provenance.Valid() {
		return Package{}, fmt.Errorf(
			"semantic normalized package requires identity and provenance",
		)
	}
	out := Package{
		id: id, provenance: provenance,
		identities:   stores.identities.table,
		authorities:  stores.authorities,
		definitions:  stores.definitions,
		resolutions:  stores.resolutions,
		declarations: stores.declarations,
		bindings:     stores.bindings,
		types:        stores.types,
		witnesses:    stores.witnesses,
		operations:   stores.operations,
		unsupported:  stores.unsupported,
	}
	out.operationView = newPackageOperationProjection(
		out.operations,
		out.identities,
	)
	if err := validateAdmittedNormalizedPackageStorage(out); err != nil {
		return Package{}, err
	}
	memberTargets, err := deriveMemberTargetCensus(out)
	if err != nil {
		return Package{}, err
	}
	out.memberTargets = memberTargets
	return out, nil
}

func (pkg Package) ID() identity.PackageID { return pkg.id }
func (pkg Package) Provenance() PackageProvenance {
	return pkg.provenance
}
func (pkg Package) DefinitionCount() int {
	return len(pkg.definitions.records)
}
func (pkg Package) ResolutionCount() int { return len(pkg.resolutions.records) }
func (pkg Package) DeclarationCount() int {
	return len(pkg.declarations.records)
}
func (pkg Package) BindingCount() int {
	return len(pkg.bindings.records)
}
func (pkg Package) TypeCount() int {
	return len(pkg.types.records)
}
func (pkg Package) OperationCount() int { return len(pkg.operations.records) }
func (pkg Package) UnsupportedCount() int {
	return len(pkg.unsupported.records)
}

type Model struct {
	projections    []packageProjection
	checker        *CheckerStore
	provider       *ProviderArtifact
	projectionGate sync.Mutex
	mu             sync.Mutex
	readStats      ProjectionReadStats
	resident       int
	closed         bool
}

type ProjectionReadStats struct {
	PackageLoads        int
	MixedPackageLoads   int
	MaxPackagesResident int
}

func (model *Model) PackageCount() int {
	if model == nil {
		return 0
	}
	return len(model.projections)
}

func (model *Model) VisitPackage(
	packageID identity.PackageID,
	visit func(Package) error,
) error {
	if model == nil || packageID.IsZero() || visit == nil {
		return fmt.Errorf(
			"semantic model package visit requires model, package, and visitor",
		)
	}
	index := searchCanonical(
		model.projections,
		func(projection packageProjection) identity.PackageID {
			return projection.id
		},
		packageID,
	)
	if index == len(model.projections) ||
		model.projections[index].id != packageID {
		return fmt.Errorf(
			"semantic model package %s is absent", packageID,
		)
	}
	projection := model.projections[index]
	model.projectionGate.Lock()
	defer model.projectionGate.Unlock()
	if err := model.beginProjection(projection); err != nil {
		return err
	}
	defer model.endProjection()
	if projection.local {
		if projection.certified {
			return visitMixedProjection(
				model.checker,
				model.provider,
				projection,
				visit,
			)
		}
		return model.checker.VisitPackage(
			projection.id,
			func(local Package) error {
				pkg, err := projection.completeLocal(local)
				if err != nil {
					return err
				}
				return visit(pkg)
			},
		)
	}
	return model.provider.VisitPackage(
		projection.id,
		func(provider Package) error {
			pkg, err := projection.completeProvider(provider)
			if err != nil {
				return err
			}
			return visit(pkg)
		},
	)
}

func (model *Model) beginProjection(
	projection packageProjection,
) error {
	model.mu.Lock()
	defer model.mu.Unlock()
	if model.closed {
		return fmt.Errorf("semantic model is closed")
	}
	model.readStats.PackageLoads++
	if projection.local && projection.certified {
		model.readStats.MixedPackageLoads++
	}
	model.resident++
	if model.resident > model.readStats.MaxPackagesResident {
		model.readStats.MaxPackagesResident = model.resident
	}
	return nil
}

func (model *Model) endProjection() {
	model.mu.Lock()
	defer model.mu.Unlock()
	model.resident--
}

func (model *Model) ensureOpen() error {
	model.mu.Lock()
	defer model.mu.Unlock()
	if model.closed {
		return fmt.Errorf("semantic model is closed")
	}
	return nil
}

func (model *Model) VisitPackages(
	visit func(Package) error,
) error {
	if model == nil || visit == nil {
		return fmt.Errorf(
			"semantic model package visit requires model and visitor",
		)
	}
	for _, projection := range model.projections {
		if err := model.VisitPackage(projection.id, visit); err != nil {
			return err
		}
	}
	return nil
}

func (model *Model) ProviderReadStats() ProviderReadStats {
	if model == nil || model.provider == nil {
		return ProviderReadStats{}
	}
	return model.provider.ReadStats()
}

func (model *Model) CheckerReadStats() CheckerStoreReadStats {
	if model == nil || model.checker == nil {
		return CheckerStoreReadStats{}
	}
	return model.checker.ReadStats()
}

func (model *Model) CheckerManifestMetrics() Metrics {
	if model == nil || model.checker == nil {
		return Metrics{}
	}
	return model.checker.ManifestMetrics()
}

func (model *Model) ProviderManifestMetrics() Metrics {
	if model == nil || model.provider == nil {
		return Metrics{}
	}
	return model.provider.ManifestMetrics()
}

func (model *Model) ProjectionReadStats() ProjectionReadStats {
	if model == nil {
		return ProjectionReadStats{}
	}
	model.mu.Lock()
	defer model.mu.Unlock()
	return model.readStats
}

func (model *Model) Close() error {
	if model == nil {
		return nil
	}
	model.projectionGate.Lock()
	defer model.projectionGate.Unlock()
	model.mu.Lock()
	if model.closed {
		model.mu.Unlock()
		return nil
	}
	model.closed = true
	model.mu.Unlock()
	if model.checker == nil {
		return nil
	}
	return model.checker.Close()
}
