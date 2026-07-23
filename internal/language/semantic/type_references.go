package semantic

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

func referencedTypeIDs(record Type) []identity.SemanticTypeID {
	spec := record.Spec()
	var references []identity.SemanticTypeID
	references = appendDeclarationOwnerType(
		references, spec.Declaration,
	)
	references = appendDeclarationOwnerType(
		references, spec.Parameter.Declaration(),
	)
	references = append(references, spec.Arguments...)
	references = append(
		references,
		spec.Underlying,
		spec.Target,
		spec.Constraint,
		spec.Element,
		spec.Key,
		spec.Signature.Receiver,
	)
	references = append(
		references, spec.Signature.ReceiverTypeParameters...,
	)
	references = append(
		references, spec.Signature.TypeParameters...,
	)
	references = append(references, spec.Signature.Parameters...)
	references = append(references, spec.Signature.Results...)
	for _, field := range spec.Fields {
		references = append(references, field.Type)
	}
	for _, method := range spec.Methods {
		references = append(references, method.Signature)
	}
	references = append(references, spec.Embeddeds...)
	for _, term := range spec.Terms {
		references = append(references, term.Type)
	}
	references = append(references, spec.Elements...)
	return references
}

func packageInputTypeRoots(
	input PackageInput,
) []identity.SemanticTypeID {
	var roots []identity.SemanticTypeID
	for _, record := range input.Definitions {
		spec := record.Spec()
		roots = append(roots, spec.Signature)
		for _, declaration := range spec.Declarations {
			roots = appendDeclarationOwnerType(
				roots, declaration,
			)
		}
	}
	for _, record := range input.Resolutions {
		roots = append(roots, record.Type())
		switch record.Kind() {
		case ResolutionStructuralOnly:
			roots = append(roots, record.Structural().Type())
			roots = appendDeclarationOwnerType(
				roots, record.Structural().Declaration(),
			)
		case ResolutionDeclaration:
			roots = appendDeclarationOwnerType(
				roots, record.Declaration(),
			)
		}
	}
	for _, record := range input.Declarations {
		roots = append(roots, record.Type())
		roots = appendDeclarationOwnerType(roots, record.ID())
	}
	for _, record := range input.Bindings {
		roots = append(roots, record.Type())
	}
	for _, record := range input.Operations {
		spec := record.Spec()
		roots = append(
			roots,
			spec.ResultType,
			spec.ExpectedType,
			spec.Selection.Receiver(),
			spec.Instance.Signature(),
		)
		roots = appendObjectOwnerType(roots, spec.Object)
		roots = appendDeclarationOwnerType(
			roots, spec.Selection.Object(),
		)
		roots = appendObjectOwnerType(
			roots, spec.Instance.Target(),
		)
		roots = append(roots, spec.Instance.Types()...)
		for _, implicit := range spec.Implicit {
			roots = append(
				roots, implicit.Source(), implicit.Target(),
			)
		}
	}
	return roots
}

func appendObjectOwnerType(
	types []identity.SemanticTypeID,
	object ObjectReference,
) []identity.SemanticTypeID {
	if object.Kind() != ObjectReferenceDeclaration {
		return types
	}
	return appendDeclarationOwnerType(types, object.Declaration())
}

func appendDeclarationOwnerType(
	types []identity.SemanticTypeID,
	declaration identity.SemanticDeclarationID,
) []identity.SemanticTypeID {
	if declaration.Form() != identity.SemanticDeclarationMember {
		return types
	}
	return append(types, declaration.OwnerType())
}

// FinalizePackageTypePool projects the exact type closure of a producer's
// transient canonical interning pool. Artifact decoding must call NewPackage
// directly so an injected, unreferenced type remains a hard admission error.
func FinalizePackageTypePool(
	input PackageInput,
) (PackageInput, error) {
	records := map[identity.SemanticTypeID]Type{}
	for _, record := range input.Types {
		if existing, duplicate := records[record.ID()]; duplicate {
			if existing.Canonical() != record.Canonical() {
				return PackageInput{}, fmt.Errorf(
					"semantic type pool collides at %s",
					record.ID(),
				)
			}
			return PackageInput{}, fmt.Errorf(
				"semantic type pool duplicates %s", record.ID(),
			)
		}
		records[record.ID()] = record
	}
	witnesses := map[identity.SemanticTypeID]TypeWitness{}
	for _, witness := range input.TypeWitnesses {
		if _, duplicate := witnesses[witness.Type()]; duplicate {
			return PackageInput{}, fmt.Errorf(
				"semantic type pool duplicates witness %s",
				witness.Type(),
			)
		}
		witnesses[witness.Type()] = witness
	}
	selected := map[identity.SemanticTypeID]bool{}
	queue := packageInputTypeRoots(input)
	for len(queue) != 0 {
		typeID := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if typeID.IsZero() || selected[typeID] {
			continue
		}
		record, present := records[typeID]
		if !present {
			return PackageInput{}, fmt.Errorf(
				"semantic package references absent type %s",
				typeID,
			)
		}
		selected[typeID] = true
		queue = append(queue, referencedTypeIDs(record)...)
	}
	typeIDs := make(
		[]identity.SemanticTypeID, 0, len(selected),
	)
	for typeID := range selected {
		typeIDs = append(typeIDs, typeID)
	}
	sort.Slice(typeIDs, func(left, right int) bool {
		return typeIDs[left].String() < typeIDs[right].String()
	})
	input.Types = make([]Type, 0, len(typeIDs))
	input.TypeWitnesses = make([]TypeWitness, 0, len(typeIDs))
	for _, typeID := range typeIDs {
		witness, present := witnesses[typeID]
		if !present {
			return PackageInput{}, fmt.Errorf(
				"semantic package type %s lacks authority witness",
				typeID,
			)
		}
		input.Types = append(input.Types, records[typeID])
		input.TypeWitnesses = append(
			input.TypeWitnesses, witness,
		)
	}
	return input, nil
}
