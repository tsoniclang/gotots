package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

type TypeParameterRole uint8

const (
	TypeParameterInvalid TypeParameterRole = iota
	TypeParameterDeclared
	TypeParameterCallable
	TypeParameterReceiver

	typeParameterRoleCount = TypeParameterReceiver
)

var typeParameterRoleNames = [typeParameterRoleCount + 1]string{
	TypeParameterDeclared: "type",
	TypeParameterCallable: "callable",
	TypeParameterReceiver: "receiver",
}

func (role TypeParameterRole) Valid() bool {
	return role > TypeParameterInvalid &&
		role <= typeParameterRoleCount
}

func (role TypeParameterRole) String() string {
	if role.Valid() {
		return typeParameterRoleNames[role]
	}
	return fmt.Sprintf(
		"semantic.TypeParameterRole(%d)", uint8(role),
	)
}

type TypeParameterOwner struct {
	declaration identity.SemanticDeclarationID
	definition  identity.DefinitionID
	role        TypeParameterRole
	ordinal     int
}

func NewTypeParameterOwner(
	declaration identity.SemanticDeclarationID,
	definition identity.DefinitionID,
	role TypeParameterRole,
	ordinal int,
) (TypeParameterOwner, error) {
	if declaration.IsZero() == definition.IsZero() ||
		!role.Valid() ||
		ordinal < 0 {
		return TypeParameterOwner{}, fmt.Errorf(
			"type-parameter owner requires exactly one declaration or definition, a closed role, and a non-negative ordinal",
		)
	}
	return TypeParameterOwner{
		declaration: declaration,
		definition:  definition,
		role:        role,
		ordinal:     ordinal,
	}, nil
}

func (owner TypeParameterOwner) IsZero() bool {
	return owner == TypeParameterOwner{}
}

func (owner TypeParameterOwner) Declaration() identity.SemanticDeclarationID {
	return owner.declaration
}

func (owner TypeParameterOwner) Definition() identity.DefinitionID {
	return owner.definition
}

func (owner TypeParameterOwner) Role() TypeParameterRole {
	return owner.role
}

func (owner TypeParameterOwner) Ordinal() int {
	return owner.ordinal
}

func (owner TypeParameterOwner) String() string {
	if owner.IsZero() {
		return ""
	}
	anchor := owner.declaration.String()
	if anchor == "" {
		anchor = owner.definition.String()
	}
	return fmt.Sprintf(
		"%s#type-parameter/%s/%d",
		anchor, owner.role, owner.ordinal,
	)
}
