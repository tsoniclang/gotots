package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

func overlayDefinitions(
	provider []DefinitionSemantics,
	local []DefinitionSemantics,
) ([]DefinitionSemantics, error) {
	out := append([]DefinitionSemantics(nil), provider...)
	index := map[identity.DefinitionID]int{}
	for position, record := range out {
		index[record.Definition()] = position
	}
	for _, record := range local {
		position, present := index[record.Definition()]
		if !present {
			return nil, missingCorroboration(
				"definition", record.Definition().String(),
			)
		}
		equal, err := canonicalWireEqual(
			encodeDefinition(out[position]),
			encodeDefinition(record),
		)
		if err != nil {
			return nil, err
		}
		if !equal {
			return nil, conflictingCorroboration(
				"definition", record.Definition().String(),
			)
		}
		out[position] = record
	}
	return out, nil
}

func overlayResolutions(
	provider []OccurrenceResolution,
	local []OccurrenceResolution,
) ([]OccurrenceResolution, error) {
	out := append([]OccurrenceResolution(nil), provider...)
	index := map[identity.OccurrenceID]int{}
	for position, record := range out {
		index[record.Occurrence()] = position
	}
	for _, record := range local {
		position, present := index[record.Occurrence()]
		if !present {
			return nil, missingCorroboration(
				"resolution", record.Occurrence().String(),
			)
		}
		equal, err := canonicalWireEqual(
			encodeResolution(out[position]),
			encodeResolution(record),
		)
		if err != nil {
			return nil, err
		}
		if !equal {
			return nil, conflictingCorroboration(
				"resolution", record.Occurrence().String(),
			)
		}
		out[position] = record
	}
	return out, nil
}

func overlayDeclarations(
	provider []Declaration,
	local []Declaration,
) ([]Declaration, error) {
	out := append([]Declaration(nil), provider...)
	index := map[identity.SemanticDeclarationID]int{}
	for position, record := range out {
		index[record.ID()] = position
	}
	for _, record := range local {
		position, present := index[record.ID()]
		if !present {
			return nil, missingCorroboration(
				"declaration", record.ID().String(),
			)
		}
		equal, err := canonicalWireEqual(
			encodeDeclaration(out[position]),
			encodeDeclaration(record),
		)
		if err != nil {
			return nil, err
		}
		if !equal {
			return nil, conflictingCorroboration(
				"declaration", record.ID().String(),
			)
		}
		out[position] = record
	}
	return out, nil
}

func overlayBindings(
	provider []Binding,
	local []Binding,
) ([]Binding, error) {
	out := append([]Binding(nil), provider...)
	index := map[identity.SemanticBindingID]int{}
	for position, record := range out {
		index[record.ID()] = position
	}
	for _, record := range local {
		position, present := index[record.ID()]
		if !present {
			return nil, missingCorroboration(
				"binding", record.ID().String(),
			)
		}
		equal, err := canonicalWireEqual(
			encodeBinding(out[position]),
			encodeBinding(record),
		)
		if err != nil {
			return nil, err
		}
		if !equal {
			return nil, conflictingCorroboration(
				"binding", record.ID().String(),
			)
		}
		out[position] = record
	}
	return out, nil
}

func overlayTypes(
	provider []Type,
	providerWitnesses []TypeWitness,
	local []Type,
	localWitnesses []TypeWitness,
) ([]Type, []TypeWitness, error) {
	out := append([]Type(nil), provider...)
	witnesses := append([]TypeWitness(nil), providerWitnesses...)
	index := map[identity.SemanticTypeID]int{}
	witnessIndex := map[identity.SemanticTypeID]int{}
	for position, record := range out {
		index[record.ID()] = position
	}
	for position, witness := range witnesses {
		witnessIndex[witness.Type()] = position
	}
	localWitnessByType := map[identity.SemanticTypeID]TypeWitness{}
	for _, witness := range localWitnesses {
		localWitnessByType[witness.Type()] = witness
	}
	for _, record := range local {
		position, present := index[record.ID()]
		if !present {
			return nil, nil, missingCorroboration(
				"type", record.ID().String(),
			)
		}
		if out[position].Canonical() != record.Canonical() {
			return nil, nil, conflictingCorroboration(
				"type", record.ID().String(),
			)
		}
		witnessPosition, present := witnessIndex[record.ID()]
		localWitness, localPresent := localWitnessByType[record.ID()]
		if !present || !localPresent {
			return nil, nil, fmt.Errorf(
				"semantic type %s lacks provider/local authority witness",
				record.ID(),
			)
		}
		out[position] = record
		witnesses[witnessPosition] = localWitness
	}
	return out, witnesses, nil
}

func overlayOperations(
	provider []Operation,
	local []Operation,
) ([]Operation, error) {
	out := append([]Operation(nil), provider...)
	index := map[identity.OperationID]int{}
	for position, record := range out {
		index[record.ID()] = position
	}
	for _, record := range local {
		position, present := index[record.ID()]
		if !present {
			return nil, missingCorroboration(
				"operation", record.ID().String(),
			)
		}
		equal, err := canonicalWireEqual(
			encodeOperation(out[position]),
			encodeOperation(record),
		)
		if err != nil {
			return nil, err
		}
		if !equal {
			return nil, conflictingCorroboration(
				"operation", record.ID().String(),
			)
		}
		out[position] = record
	}
	return out, nil
}

func overlayUnsupported(
	provider []Unsupported,
	local []Unsupported,
) ([]Unsupported, error) {
	out := append([]Unsupported(nil), provider...)
	index := map[identity.UnsupportedID]int{}
	for position, record := range out {
		index[record.ID()] = position
	}
	for _, record := range local {
		position, present := index[record.ID()]
		if !present {
			return nil, missingCorroboration(
				"unsupported", record.ID().String(),
			)
		}
		equal, err := canonicalWireEqual(
			encodeUnsupported(out[position]),
			encodeUnsupported(record),
		)
		if err != nil {
			return nil, err
		}
		if !equal {
			return nil, conflictingCorroboration(
				"unsupported", record.ID().String(),
			)
		}
		out[position] = record
	}
	return out, nil
}

func missingCorroboration(kind string, id string) error {
	return fmt.Errorf(
		"semantic provider lacks checker-corroborated %s %s",
		kind, id,
	)
}

func conflictingCorroboration(kind string, id string) error {
	return fmt.Errorf(
		"semantic checker/provider %s differs after authority removal: %s",
		kind, id,
	)
}
