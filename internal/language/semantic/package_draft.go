package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

// PackageDraft is the single-owner producer path for building one immutable
// semantic package without retaining a second complete record set. It exposes
// append-only typed methods and no backing storage.
type PackageDraft struct {
	id               identity.PackageID
	provenance       PackageProvenance
	normalized       normalizedPackageBuilder
	declarationRoots []declarationRef
	typeRoots        []typeRef
	sealed           bool
}

type PackageCapacity struct {
	Definitions  int
	Resolutions  int
	Declarations int
	Bindings     int
	Types        int
	Operations   int
	Unsupported  int
}

func (capacity PackageCapacity) valid() bool {
	return capacity.Definitions >= 0 &&
		capacity.Resolutions >= 0 &&
		capacity.Declarations >= 0 &&
		capacity.Bindings >= 0 &&
		capacity.Types >= 0 &&
		capacity.Operations >= 0 &&
		capacity.Unsupported >= 0
}

func NewPackageDraft(
	id identity.PackageID,
	provenance PackageProvenance,
	capacity PackageCapacity,
) (*PackageDraft, error) {
	if id.IsZero() || !provenance.Valid() || !capacity.valid() {
		return nil, fmt.Errorf(
			"semantic package draft requires identity, provenance, and nonnegative capacities",
		)
	}
	return &PackageDraft{
		id:         id,
		provenance: provenance,
		normalized: normalizedPackageBuilder{
			definitions: packageDefinitionBuilder{
				records: make(
					[]storedDefinition, 0, capacity.Definitions,
				),
			},
			resolutions: packageResolutionBuilder{
				records: make(
					[]storedResolution, 0, capacity.Resolutions,
				),
			},
			declarations: packageDeclarationBuilder{
				records: make(
					[]storedDeclaration, 0, capacity.Declarations,
				),
			},
			bindings: packageBindingBuilder{
				records: make(
					[]storedBinding, 0, capacity.Bindings,
				),
			},
			types: packageTypeBuilder{
				records: make(
					[]storedType, 0, capacity.Types,
				),
			},
			witnesses: packageTypeWitnessBuilder{
				records: make(
					[]storedTypeWitness, 0, capacity.Types,
				),
			},
			operations: packageOperationBuilder{
				records: make(
					[]storedOperation, 0, capacity.Operations,
				),
			},
			unsupported: packageUnsupportedBuilder{
				records: make(
					[]storedUnsupported, 0, capacity.Unsupported,
				),
			},
		},
	}, nil
}

func (draft *PackageDraft) ensureOpen() error {
	if draft == nil || draft.sealed {
		return fmt.Errorf("semantic package draft is not open")
	}
	return nil
}

func (draft *PackageDraft) AddDefinition(
	record DefinitionSemantics,
) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	draft.normalized.addDefinition(record)
	spec := record.spec
	for _, declaration := range spec.Declarations {
		draft.addDeclarationRoot(declaration)
	}
	draft.addTypeRoot(spec.Signature)
	for _, declaration := range spec.Declarations {
		draft.addDeclarationOwnerTypeRoot(declaration)
	}
	return nil
}

func (draft *PackageDraft) AddResolution(
	record OccurrenceResolution,
) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	draft.normalized.addResolution(record)
	draft.addResolutionRoots(record)
	return nil
}

func (draft *PackageDraft) AddDeclaration(record Declaration) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	draft.normalized.addDeclaration(record)
	draft.addTypeRoot(record.typeID)
	draft.addDeclarationOwnerTypeRoot(record.id)
	return nil
}

func (draft *PackageDraft) AddBinding(record Binding) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	draft.normalized.addBinding(record)
	draft.addTypeRoot(record.typeID)
	return nil
}

func (draft *PackageDraft) AddType(
	record Type,
	witness TypeWitness,
) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	draft.normalized.addType(record)
	draft.normalized.addTypeWitness(witness)
	return nil
}

func (draft *PackageDraft) VisitTypeRoots(
	visit func(identity.SemanticTypeID) error,
) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	if visit == nil {
		return fmt.Errorf(
			"semantic package draft type-root visitor is absent",
		)
	}
	identities := newPackageIdentityProjection(
		draft.normalized.identities.projectionTable(),
	)
	for _, reference := range draft.typeRoots {
		typeID := identities.typeID(reference)
		if typeID.IsZero() {
			continue
		}
		if err := visit(typeID); err != nil {
			return err
		}
	}
	return nil
}

func (draft *PackageDraft) VisitDeclarationRoots(
	visit func(identity.SemanticDeclarationID) error,
) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	if visit == nil {
		return fmt.Errorf(
			"semantic package draft declaration-root visitor is absent",
		)
	}
	identities := newPackageIdentityProjection(
		draft.normalized.identities.projectionTable(),
	)
	for _, reference := range draft.declarationRoots {
		declaration := identities.declaration(reference)
		if declaration.IsZero() {
			continue
		}
		if err := visit(declaration); err != nil {
			return err
		}
	}
	return nil
}

func (draft *PackageDraft) AddOperation(spec OperationSpec) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	if _, err := validateOperationSpec(spec); err != nil {
		return err
	}
	draft.normalized.addOperationSpec(spec)
	draft.addOperationRoots(spec)
	return nil
}

func (draft *PackageDraft) addDeclarationRoot(
	declaration identity.SemanticDeclarationID,
) {
	if declaration.IsZero() {
		return
	}
	draft.declarationRoots = append(
		draft.declarationRoots,
		draft.normalized.identities.declaration(declaration),
	)
}

func (draft *PackageDraft) addTypeRoot(typeID identity.SemanticTypeID) {
	if typeID.IsZero() {
		return
	}
	draft.typeRoots = append(
		draft.typeRoots,
		draft.normalized.identities.typeID(typeID),
	)
}

func (draft *PackageDraft) addDeclarationOwnerTypeRoot(
	declaration identity.SemanticDeclarationID,
) {
	if declaration.Form() ==
		identity.SemanticDeclarationMember {
		draft.addTypeRoot(declaration.OwnerType())
	}
}

func (draft *PackageDraft) addObjectRoots(object ObjectReference) {
	if object.kind != ObjectReferenceDeclaration {
		return
	}
	draft.addDeclarationRoot(object.declaration)
	draft.addDeclarationOwnerTypeRoot(object.declaration)
}

func (draft *PackageDraft) addResolutionRoots(
	record OccurrenceResolution,
) {
	draft.addTypeRoot(record.typeID)
	switch record.kind {
	case ResolutionStructuralOnly:
		draft.addDeclarationRoot(record.structural.declaration)
		draft.addTypeRoot(record.structural.typeID)
		draft.addDeclarationOwnerTypeRoot(
			record.structural.declaration,
		)
	case ResolutionDeclaration:
		draft.addDeclarationRoot(record.declaration)
		draft.addDeclarationOwnerTypeRoot(record.declaration)
	}
}

func (draft *PackageDraft) addOperationRoots(spec OperationSpec) {
	draft.addObjectRoots(spec.Object)
	draft.addDeclarationRoot(spec.Selection.object)
	draft.addDeclarationOwnerTypeRoot(spec.Selection.object)
	draft.addObjectRoots(spec.Instance.target)
	draft.addTypeRoot(spec.ResultType)
	draft.addTypeRoot(spec.ExpectedType)
	draft.addTypeRoot(spec.Selection.receiver)
	draft.addTypeRoot(spec.Instance.signature)
	for _, typeID := range spec.Instance.types {
		draft.addTypeRoot(typeID)
	}
	for _, implicit := range spec.Implicit {
		draft.addTypeRoot(implicit.source)
		draft.addTypeRoot(implicit.target)
	}
}

func (draft *PackageDraft) AddUnsupported(record Unsupported) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	draft.normalized.addUnsupported(record)
	return nil
}

func (draft *PackageDraft) ResolutionCount() int {
	if draft == nil {
		return 0
	}
	return len(draft.normalized.resolutions.records)
}

func (draft *PackageDraft) OperationCount() int {
	if draft == nil {
		return 0
	}
	return len(draft.normalized.operations.records)
}

func (draft *PackageDraft) UnsupportedCount() int {
	if draft == nil {
		return 0
	}
	return len(draft.normalized.unsupported.records)
}

func (draft *PackageDraft) SealProducer() (Package, error) {
	return draft.seal()
}

func (draft *PackageDraft) sealArtifact() (Package, error) {
	return draft.seal()
}

func (draft *PackageDraft) seal() (Package, error) {
	if err := draft.ensureOpen(); err != nil {
		return Package{}, err
	}
	pkg, err := newPackageFromBuilder(
		draft.id, draft.provenance, &draft.normalized,
	)
	if err != nil {
		return Package{}, err
	}
	draft.id = identity.PackageID{}
	draft.provenance = ProvenanceInvalid
	draft.normalized = normalizedPackageBuilder{}
	draft.declarationRoots = nil
	draft.typeRoots = nil
	draft.sealed = true
	return pkg, nil
}
