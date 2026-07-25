package frontend

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type semanticClosure struct {
	builder              *packageBuilder
	ownedDeclarations    map[identity.SemanticDeclarationID]bool
	selectedDeclarations map[identity.SemanticDeclarationID]bool
	selectedTypes        map[identity.SemanticTypeID]bool
	declarationQueue     []identity.SemanticDeclarationID
	typeQueue            []identity.SemanticTypeID
	declarationCursor    int
	typeCursor           int
}

func (builder *packageBuilder) materializeSemanticClosure() (
	[]identity.SemanticDeclarationID,
	int,
	error,
) {
	owned, err := builder.objects.ownedDeclarationIDs()
	if err != nil {
		return nil, 0, err
	}
	closure := semanticClosure{
		builder:              builder,
		ownedDeclarations:    make(map[identity.SemanticDeclarationID]bool, len(owned)),
		selectedDeclarations: map[identity.SemanticDeclarationID]bool{},
		selectedTypes:        map[identity.SemanticTypeID]bool{},
	}
	for _, declaration := range owned {
		closure.ownedDeclarations[declaration] = true
	}
	if builder.stage.allLocal ||
		!packageUsesCertifiedSemantics(
			builder.stage.plan,
			builder.input.loaded,
		) {
		for _, declaration := range owned {
			closure.requireDeclaration(declaration)
		}
	} else if err := builder.draft.VisitDeclarationRoots(
		func(declaration identity.SemanticDeclarationID) error {
			closure.requireDeclaration(declaration)
			return nil
		},
	); err != nil {
		return nil, 0, err
	}
	if err := builder.draft.VisitTypeRoots(
		func(typeID identity.SemanticTypeID) error {
			closure.requireType(typeID)
			return nil
		},
	); err != nil {
		return nil, 0, err
	}
	if err := closure.materialize(); err != nil {
		return nil, 0, err
	}
	declarations := make(
		[]identity.SemanticDeclarationID,
		0,
		len(closure.selectedDeclarations),
	)
	for declaration := range closure.selectedDeclarations {
		declarations = append(declarations, declaration)
	}
	sort.Slice(declarations, func(left, right int) bool {
		return declarations[left].Compare(declarations[right]) < 0
	})
	typeCount, err := closure.transferTypes()
	if err != nil {
		return nil, 0, err
	}
	return declarations, typeCount, nil
}

func (closure *semanticClosure) requireDeclaration(
	declaration identity.SemanticDeclarationID,
) {
	if declaration.IsZero() ||
		!closure.ownedDeclarations[declaration] ||
		closure.selectedDeclarations[declaration] {
		return
	}
	closure.selectedDeclarations[declaration] = true
	closure.declarationQueue = append(
		closure.declarationQueue, declaration,
	)
}

func (closure *semanticClosure) requireType(
	typeID identity.SemanticTypeID,
) {
	if typeID.IsZero() || closure.selectedTypes[typeID] {
		return
	}
	closure.selectedTypes[typeID] = true
	closure.typeQueue = append(closure.typeQueue, typeID)
}

func (closure *semanticClosure) materialize() error {
	for closure.declarationCursor < len(closure.declarationQueue) ||
		closure.typeCursor < len(closure.typeQueue) {
		for closure.declarationCursor < len(closure.declarationQueue) {
			declaration := closure.declarationQueue[closure.declarationCursor]
			closure.declarationCursor++
			record, present, err :=
				closure.builder.objects.ownedDeclarationRecord(
					declaration,
				)
			if err != nil {
				return err
			}
			if !present {
				return fmt.Errorf(
					"semantic closure declaration %s has no owned record",
					declaration,
				)
			}
			if err := closure.builder.draft.AddDeclaration(
				record,
			); err != nil {
				return err
			}
			closure.requireType(record.Type())
		}
		if err := closure.builder.types.finish(); err != nil {
			return err
		}
		for closure.typeCursor < len(closure.typeQueue) {
			typeID := closure.typeQueue[closure.typeCursor]
			closure.typeCursor++
			record, present := closure.builder.types.records[typeID]
			if !present {
				return fmt.Errorf(
					"semantic package references absent type %s",
					typeID,
				)
			}
			if err := record.VisitReferences(
				func(reference identity.SemanticTypeID) error {
					closure.requireType(reference)
					return nil
				},
			); err != nil {
				return err
			}
			if err := record.VisitDeclarationReferences(
				func(declaration identity.SemanticDeclarationID) error {
					closure.requireDeclaration(declaration)
					return nil
				},
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (closure *semanticClosure) transferTypes() (int, error) {
	typeIDs := make(
		[]identity.SemanticTypeID,
		0,
		len(closure.selectedTypes),
	)
	for typeID := range closure.selectedTypes {
		typeIDs = append(typeIDs, typeID)
	}
	sort.Slice(typeIDs, func(left, right int) bool {
		return typeIDs[left].Compare(typeIDs[right]) < 0
	})
	for _, typeID := range typeIDs {
		record, present := closure.builder.types.records[typeID]
		if !present {
			return 0, fmt.Errorf(
				"semantic selected type %s is absent", typeID,
			)
		}
		witness, err := semantic.NewTypeWitness(
			closure.builder.input.id,
			typeID,
			closure.builder.input.authority,
		)
		if err != nil {
			return 0, err
		}
		if err := closure.builder.draft.AddType(
			record, witness,
		); err != nil {
			return 0, err
		}
		delete(closure.builder.types.records, typeID)
	}
	closure.builder.types.records = nil
	return len(typeIDs), nil
}
