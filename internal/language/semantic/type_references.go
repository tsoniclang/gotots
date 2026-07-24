package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

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
