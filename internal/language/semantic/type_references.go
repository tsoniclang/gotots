package semantic

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

func referencedTypeIDs(record Type) []identity.SemanticTypeID {
	var references []identity.SemanticTypeID
	_ = record.VisitReferences(func(typeID identity.SemanticTypeID) error {
		references = append(references, typeID)
		return nil
	})
	return references
}

func (record Type) VisitReferences(
	visit func(identity.SemanticTypeID) error,
) error {
	if visit == nil {
		return fmt.Errorf("semantic type reference visitor is absent")
	}
	emit := func(typeID identity.SemanticTypeID) error {
		if typeID.IsZero() {
			return nil
		}
		return visit(typeID)
	}
	emitDeclarationOwner := func(
		declaration identity.SemanticDeclarationID,
	) error {
		if declaration.Form() !=
			identity.SemanticDeclarationMember {
			return nil
		}
		return emit(declaration.OwnerType())
	}
	spec := record.spec
	if err := emitDeclarationOwner(spec.Declaration); err != nil {
		return err
	}
	if err := emitDeclarationOwner(
		spec.Parameter.Declaration(),
	); err != nil {
		return err
	}
	singles := [...]identity.SemanticTypeID{
		spec.Underlying,
		spec.Target,
		spec.Constraint,
		spec.Element,
		spec.Key,
		spec.Signature.Receiver,
	}
	for _, typeID := range singles {
		if err := emit(typeID); err != nil {
			return err
		}
	}
	lists := [][]identity.SemanticTypeID{
		spec.Arguments,
		spec.Signature.ReceiverTypeParameters,
		spec.Signature.TypeParameters,
		spec.Signature.Parameters,
		spec.Signature.Results,
		spec.Embeddeds,
		spec.Elements,
	}
	for _, values := range lists {
		for _, typeID := range values {
			if err := emit(typeID); err != nil {
				return err
			}
		}
	}
	for _, field := range spec.Fields {
		if err := emit(field.Type); err != nil {
			return err
		}
	}
	for _, method := range spec.Methods {
		if err := emit(method.Signature); err != nil {
			return err
		}
	}
	for _, term := range spec.Terms {
		if err := emit(term.Type); err != nil {
			return err
		}
	}
	return nil
}

func (record Type) VisitDeclarationReferences(
	visit func(identity.SemanticDeclarationID) error,
) error {
	if visit == nil {
		return fmt.Errorf(
			"semantic type declaration-reference visitor is absent",
		)
	}
	declarations := [...]identity.SemanticDeclarationID{
		record.spec.Declaration,
		record.spec.Parameter.Declaration(),
	}
	for _, declaration := range declarations {
		if declaration.IsZero() {
			continue
		}
		if err := visit(declaration); err != nil {
			return err
		}
	}
	return nil
}

func appendResolutionDeclarationRoots(
	roots []identity.SemanticDeclarationID,
	record OccurrenceResolution,
) []identity.SemanticDeclarationID {
	switch record.Kind() {
	case ResolutionStructuralOnly:
		declaration := record.Structural().Declaration()
		if !declaration.IsZero() {
			roots = append(roots, declaration)
		}
	case ResolutionDeclaration:
		roots = append(roots, record.Declaration())
	}
	return roots
}

func appendOperationDeclarationRoots(
	roots []identity.SemanticDeclarationID,
	record Operation,
) []identity.SemanticDeclarationID {
	spec := record.spec
	roots = appendObjectDeclarationRoot(roots, spec.Object)
	if !spec.Selection.IsZero() {
		roots = append(
			roots, spec.Selection.Object(),
		)
	}
	if !spec.Instance.IsZero() {
		roots = appendObjectDeclarationRoot(
			roots, spec.Instance.Target(),
		)
	}
	return roots
}

func appendObjectDeclarationRoot(
	roots []identity.SemanticDeclarationID,
	object ObjectReference,
) []identity.SemanticDeclarationID {
	if object.Kind() == ObjectReferenceDeclaration {
		roots = append(roots, object.Declaration())
	}
	return roots
}

func packageInputTypeRoots(
	input PackageInput,
) []identity.SemanticTypeID {
	var roots []identity.SemanticTypeID
	for _, record := range input.Definitions {
		spec := record.spec
		roots = append(roots, spec.Signature)
		for _, declaration := range spec.Declarations {
			roots = appendDeclarationOwnerType(
				roots, declaration,
			)
		}
	}
	for _, record := range input.Resolutions {
		roots = appendResolutionTypeRoots(roots, record)
	}
	for _, record := range input.Declarations {
		roots = append(roots, record.Type())
		roots = appendDeclarationOwnerType(roots, record.ID())
	}
	for _, record := range input.Bindings {
		roots = append(roots, record.Type())
	}
	for _, record := range input.Operations {
		roots = appendOperationTypeRoots(roots, record)
	}
	return roots
}

func appendResolutionTypeRoots(
	roots []identity.SemanticTypeID,
	record OccurrenceResolution,
) []identity.SemanticTypeID {
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
	return roots
}

func appendOperationTypeRoots(
	roots []identity.SemanticTypeID,
	record Operation,
) []identity.SemanticTypeID {
	spec := record.spec
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
	roots = append(roots, spec.Instance.types...)
	for _, implicit := range spec.Implicit {
		roots = append(
			roots, implicit.Source(), implicit.Target(),
		)
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
	return finalizePackageTypePool(
		input, packageInputTypeRoots(input),
	)
}

func finalizePackageTypePool(
	input PackageInput,
	roots []identity.SemanticTypeID,
) (PackageInput, error) {
	records := map[identity.SemanticTypeID]Type{}
	for _, record := range input.Types {
		if existing, duplicate := records[record.ID()]; duplicate {
			if !existing.Equal(record) {
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
	queue := append(
		[]identity.SemanticTypeID(nil), roots...,
	)
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
		return typeIDs[left].Compare(typeIDs[right]) < 0
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
