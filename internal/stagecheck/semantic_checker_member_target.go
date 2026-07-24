package stagecheck

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

func (verifier *checkerSemanticVerifier) verifyMemberDeclarationOccurrence(
	occurrence structure.Occurrence,
	id identity.SemanticDeclarationID,
	object types.Object,
) error {
	if id.Form() != identity.SemanticDeclarationMember || object == nil {
		return fmt.Errorf(
			"member declaration occurrence requires member identity and checker object",
		)
	}
	owner, err := verifier.independentMemberDeclarationOwner(
		occurrence, object,
	)
	if err != nil {
		return err
	}
	if err := verifier.types.verify(id.OwnerType(), owner); err != nil {
		return fmt.Errorf("member declaration owner: %w", err)
	}
	ordinal, err := independentMemberOrdinal(object, owner)
	if err != nil {
		return err
	}
	if err := verifier.verifyMemberDeclarationIdentity(
		id, object, owner, ordinal,
	); err != nil {
		return err
	}
	target, present := verifier.actual.ResolveDeclarationTarget(id)
	if !present {
		return fmt.Errorf(
			"member declaration target %s is absent", id,
		)
	}
	return verifier.verifyMemberTargetPayload(target, object, ordinal)
}

func (
	verifier *checkerSemanticVerifier,
) independentMemberDeclarationOwner(
	occurrence structure.Occurrence,
	object types.Object,
) (types.Type, error) {
	if function, method := object.(*types.Func); method {
		origin := function.Origin()
		signature, _ := origin.Type().(*types.Signature)
		if signature != nil && signature.Recv() != nil {
			return independentOriginMemberOwner(
				signature.Recv().Type(),
			), nil
		}
	}
	parent := occurrence.Parent()
	for !parent.IsZero() {
		node, present := verifier.index.OccurrenceNode(parent)
		if !present {
			return nil, fmt.Errorf(
				"member declaration parent %s has no transient node",
				parent,
			)
		}
		var aggregate types.Type
		switch typed := node.(type) {
		case *ast.StructType:
			if value, present := verifier.view.TypeOf(typed); present &&
				value.Type != nil {
				aggregate = value.Type
			}
		case *ast.InterfaceType:
			if value, present := verifier.view.TypeOf(typed); present &&
				value.Type != nil {
				aggregate = value.Type
			}
		}
		if aggregate != nil {
			aggregateOccurrence, present :=
				verifier.expected.occurrences.get(parent)
			if !present {
				return nil, fmt.Errorf(
					"member aggregate %s has no structural occurrence",
					parent,
				)
			}
			declarationOccurrence, direct :=
				verifier.expected.occurrences.get(
					aggregateOccurrence.Parent(),
				)
			if direct &&
				aggregateOccurrence.Role() ==
					catalog.RoleTypeExpression {
				declarationNode, nodePresent :=
					verifier.index.OccurrenceNode(
						declarationOccurrence.ID(),
					)
				if typeSpec, named :=
					declarationNode.(*ast.TypeSpec); nodePresent &&
					named {
					declaration, defined :=
						verifier.view.DefOf(typeSpec.Name)
					if defined && declaration != nil {
						return independentOriginMemberOwner(
							declaration.Type(),
						), nil
					}
				}
			}
			return independentOriginMemberOwner(aggregate), nil
		}
		current, present := verifier.expected.occurrences.get(parent)
		if !present {
			return nil, fmt.Errorf(
				"member declaration parent %s has no structural occurrence",
				parent,
			)
		}
		parent = current.Parent()
	}
	return nil, fmt.Errorf(
		"member declaration %s has no aggregate owner",
		object.Name(),
	)
}

func independentOriginMemberOwner(typ types.Type) types.Type {
	current := typ
	for {
		current = stripCheckerPointer(current)
		target := types.Unalias(current)
		if target == current {
			break
		}
		current = target
	}
	if named, ok := current.(*types.Named); ok {
		return named.Origin()
	}
	return current
}

func independentMemberOrdinal(
	object types.Object,
	owner types.Type,
) (int, error) {
	field, isField := object.(*types.Var)
	if !isField || !field.IsField() {
		return 0, nil
	}
	structure, ok := types.Unalias(
		stripCheckerPointer(owner),
	).Underlying().(*types.Struct)
	if !ok {
		return 0, fmt.Errorf(
			"field %s has non-struct owner %s",
			field.Name(),
			types.TypeString(owner, nil),
		)
	}
	for ordinal := 0; ordinal < structure.NumFields(); ordinal++ {
		if structure.Field(ordinal) == field {
			return ordinal, nil
		}
	}
	return 0, fmt.Errorf(
		"field %s is absent from owner %s",
		field.Name(),
		types.TypeString(owner, nil),
	)
}

func (verifier *checkerSemanticVerifier) verifyMemberTargetPayload(
	target semantic.DeclarationTarget,
	object types.Object,
	ordinal int,
) error {
	var pkg identity.PackageID
	if !object.Exported() {
		if object.Pkg() == nil {
			return fmt.Errorf(
				"unexported member %s has no package", object.Name(),
			)
		}
		pkg = verifier.types.packageByPath[object.Pkg().Path()]
	}
	switch target.Kind() {
	case semantic.DeclarationTargetField:
		field, present := target.Field()
		checker, isField := object.(*types.Var)
		if !present || !isField || !checker.IsField() ||
			field.Name != checker.Name() ||
			field.Package != pkg ||
			field.Embedded != checker.Embedded() ||
			field.Ordinal != ordinal {
			return fmt.Errorf(
				"field target %s metadata differs",
				target.ID(),
			)
		}
		return verifier.types.verify(field.Type, checker.Type())
	case semantic.DeclarationTargetMethod:
		method, present := target.Method()
		checker, isMethod := object.(*types.Func)
		if !present || !isMethod ||
			method.Name != checker.Name() ||
			method.Package != pkg {
			return fmt.Errorf(
				"method target %s metadata differs",
				target.ID(),
			)
		}
		signature, _ := checker.Origin().Type().(*types.Signature)
		if signature == nil {
			return fmt.Errorf(
				"method target %s has no checker signature",
				target.ID(),
			)
		}
		return verifier.types.verifyMethodSignature(
			method.Signature, signature,
		)
	default:
		return fmt.Errorf(
			"member target %s has non-member kind",
			target.ID(),
		)
	}
}
