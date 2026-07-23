package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

// PackageDraft is the single-owner producer path for building one immutable
// semantic package without retaining a second complete record set. It exposes
// append-only typed methods and no backing storage.
type PackageDraft struct {
	input  PackageInput
	sealed bool
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
		input: PackageInput{
			ID:         id,
			Provenance: provenance,
			Definitions: make(
				[]DefinitionSemantics, 0, capacity.Definitions,
			),
			Resolutions: make(
				[]OccurrenceResolution, 0, capacity.Resolutions,
			),
			Declarations: make(
				[]Declaration, 0, capacity.Declarations,
			),
			Bindings: make(
				[]Binding, 0, capacity.Bindings,
			),
			Types: make(
				[]Type, 0, capacity.Types,
			),
			TypeWitnesses: make(
				[]TypeWitness, 0, capacity.Types,
			),
			Operations: make(
				[]Operation, 0, capacity.Operations,
			),
			Unsupported: make(
				[]Unsupported, 0, capacity.Unsupported,
			),
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
	draft.input.Definitions = append(draft.input.Definitions, record)
	return nil
}

func (draft *PackageDraft) AddResolution(
	record OccurrenceResolution,
) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	draft.input.Resolutions = append(draft.input.Resolutions, record)
	return nil
}

func (draft *PackageDraft) AddDeclaration(record Declaration) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	draft.input.Declarations = append(draft.input.Declarations, record)
	return nil
}

func (draft *PackageDraft) AddBinding(record Binding) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	draft.input.Bindings = append(draft.input.Bindings, record)
	return nil
}

func (draft *PackageDraft) AddType(
	record Type,
	witness TypeWitness,
) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	draft.input.Types = append(draft.input.Types, record)
	draft.input.TypeWitnesses = append(
		draft.input.TypeWitnesses, witness,
	)
	return nil
}

func (draft *PackageDraft) AddOperation(record Operation) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	draft.input.Operations = append(draft.input.Operations, record)
	return nil
}

func (draft *PackageDraft) AddUnsupported(record Unsupported) error {
	if err := draft.ensureOpen(); err != nil {
		return err
	}
	draft.input.Unsupported = append(
		draft.input.Unsupported, record,
	)
	return nil
}

func (draft *PackageDraft) ResolutionCount() int {
	if draft == nil {
		return 0
	}
	return len(draft.input.Resolutions)
}

func (draft *PackageDraft) OperationCount() int {
	if draft == nil {
		return 0
	}
	return len(draft.input.Operations)
}

func (draft *PackageDraft) UnsupportedCount() int {
	if draft == nil {
		return 0
	}
	return len(draft.input.Unsupported)
}

func (draft *PackageDraft) SealProducer() (Package, error) {
	if err := draft.ensureOpen(); err != nil {
		return Package{}, err
	}
	input, err := FinalizePackageTypePool(draft.input)
	if err != nil {
		return Package{}, err
	}
	pkg, err := newPackage(input, false)
	if err != nil {
		return Package{}, err
	}
	draft.input = PackageInput{}
	draft.sealed = true
	return pkg, nil
}
