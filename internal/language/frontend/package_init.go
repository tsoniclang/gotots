package frontend

import (
	"fmt"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	contract "github.com/tsoniclang/gotots/internal/scope/contract"
)

func (builder *packageBuilder) buildImplicitOperations() error {
	for definition, region := range builder.input.regions {
		if !definition.ImplicitOp().Valid() {
			continue
		}
		if definition.ImplicitOp() !=
			identity.ImplicitDefinitionPackageInit {
			return fmt.Errorf(
				"implicit definition %s has no semantic operation owner",
				definition,
			)
		}
		if err := builder.buildPackageInitialization(
			definition, region,
		); err != nil {
			return err
		}
	}
	return nil
}

func (builder *packageBuilder) buildPackageInitialization(
	definition identity.DefinitionID,
	region executable.Region,
) error {
	implicit := region.ImplicitOperations()
	if len(implicit) != 1 ||
		implicit[0].Kind() !=
			executable.ImplicitOperationCoordinatePackageInitialization {
		return fmt.Errorf(
			"package initialization %s has invalid Stage-1 operations",
			definition,
		)
	}
	id, err := identity.NewImplicitOperationID(
		definition,
		identity.ImplicitDefinitionPackageInit,
		0,
	)
	if err != nil {
		return err
	}
	operands, definitions, effects, err :=
		builder.packageInitializationSequence()
	if err != nil {
		return err
	}
	operation, err := semantic.NewOperation(semantic.OperationSpec{
		ID:          id,
		Kind:        semantic.OperationPackageInitialization,
		Mode:        semantic.ValueModeNone,
		Arity:       semantic.ResultArityZero,
		Place:       semantic.PlaceNone,
		Object:      semantic.NoObjectReference(),
		Operands:    operands,
		Definitions: definitions,
		Implicit:    effects,
	})
	if err != nil {
		return err
	}
	return builder.draft.AddOperation(operation)
}

func (builder *packageBuilder) packageInitializationSequence() (
	[]identity.OccurrenceID,
	[]identity.DefinitionID,
	[]semantic.ImplicitOperation,
	error,
) {
	view := builder.input.loaded.CheckerView()
	if view == nil {
		return nil, nil, nil, fmt.Errorf(
			"package initialization %s has no checker initialization order",
			builder.input.id,
		)
	}
	var (
		operands    []identity.OccurrenceID
		definitions []identity.DefinitionID
		effects     []semantic.ImplicitOperation
	)
	seenDefinitions := map[identity.DefinitionID]bool{}
	initialized := map[*types.Var]bool{}
	ordinal := 0
	for _, entry := range view.InitOrder() {
		entryDefinitions := map[identity.DefinitionID]bool{}
		for _, variable := range entry.Vars {
			if variable.Name() == "_" {
				continue
			}
			definition := builder.objects.definitionByObject[variable]
			if definition.IsZero() {
				return nil, nil, nil, fmt.Errorf(
					"package initializer variable %s has no canonical definition",
					variable.Name(),
				)
			}
			entryDefinitions[definition] = true
		}
		occurrence, occurrencePresent := builder.input.index.
			OccurrenceID(entry.Rhs)
		record := builder.input.occurrence(occurrence)
		if occurrencePresent && record == nil {
			return nil, nil, nil, fmt.Errorf(
				"package initializer occurrence %s is absent from semantic input",
				occurrence,
			)
		}
		if record != nil {
			entryDefinitions[record.owner] = true
		}
		fullDefinitions := map[identity.DefinitionID]bool{}
		for definition := range entryDefinitions {
			selection, present := builder.input.selections[definition]
			if !present {
				return nil, nil, nil, fmt.Errorf(
					"package initializer definition %s has no selection",
					definition,
				)
			}
			_, hasRegion := builder.input.regions[definition]
			if hasRegion {
				fullDefinitions[definition] = true
			}
			if hasRegion !=
				(selection.Depth() ==
					contract.DepthFullSemantic) {
				return nil, nil, nil, fmt.Errorf(
					"package initializer definition %s depth and region disagree",
					definition,
				)
			}
		}
		if len(fullDefinitions) == 0 {
			if len(entryDefinitions) == 0 {
				return nil, nil, nil, fmt.Errorf(
					"package initializer expression %T has no semantic definition",
					entry.Rhs,
				)
			}
			continue
		}
		if len(fullDefinitions) != len(entryDefinitions) {
			return nil, nil, nil, fmt.Errorf(
				"package initializer mixes %d full and %d non-full definitions",
				len(fullDefinitions),
				len(entryDefinitions)-len(fullDefinitions),
			)
		}
		if !occurrencePresent {
			return nil, nil, nil, fmt.Errorf(
				"package initializer expression %T has no Stage-1 occurrence",
				entry.Rhs,
			)
		}
		operands = append(operands, occurrence)
		if !seenDefinitions[record.owner] {
			seenDefinitions[record.owner] = true
			definitions = append(definitions, record.owner)
		}
		effect, err := semantic.NewImplicitOperation(
			catalog.ImplicitInitialization,
			occurrence,
			ordinal,
			identity.SemanticTypeID{},
			identity.SemanticTypeID{},
		)
		if err != nil {
			return nil, nil, nil, err
		}
		ordinal++
		effects = append(effects, effect)
		for _, variable := range entry.Vars {
			initialized[variable] = true
		}
	}
	zeroOrdinal := 0
	type zeroCandidate struct {
		variable *types.Var
		source   identity.OccurrenceID
	}
	var zeroCandidates []zeroCandidate
	for object, source := range builder.objects.sourceByObject {
		variable, ok := object.(*types.Var)
		if !ok ||
			variable.IsField() ||
			variable.Pkg() != builder.input.loaded.Types() ||
			variable.Parent() != variable.Pkg().Scope() ||
			initialized[variable] ||
			source.IsZero() {
			continue
		}
		zeroCandidates = append(zeroCandidates, zeroCandidate{
			variable: variable,
			source:   source,
		})
	}
	sort.Slice(zeroCandidates, func(left, right int) bool {
		return zeroCandidates[left].source.Compare(
			zeroCandidates[right].source,
		) < 0
	})
	for _, candidate := range zeroCandidates {
		variable := candidate.variable
		source := candidate.source
		target, err := builder.types.build(variable.Type())
		if err != nil {
			return nil, nil, nil, err
		}
		effect, err := semantic.NewImplicitOperation(
			catalog.ImplicitZeroing,
			source,
			zeroOrdinal,
			identity.SemanticTypeID{},
			target,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		zeroOrdinal++
		effects = append(effects, effect)
	}
	if len(operands) > 1 {
		effect, err := semantic.NewImplicitOperation(
			catalog.ImplicitEvaluationOrder,
			operands[0],
			0,
			identity.SemanticTypeID{},
			identity.SemanticTypeID{},
		)
		if err != nil {
			return nil, nil, nil, err
		}
		effects = append(effects, effect)
	}
	return operands, definitions, effects, nil
}
