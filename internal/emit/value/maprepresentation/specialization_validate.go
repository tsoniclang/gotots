package maprepresentation

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func validateSpecialization(
	role api.Role,
	members []tsgo.ClassElement,
	capabilities SpecializationCapabilities,
) error {
	names, err := specializationNames()
	if err != nil {
		return err
	}
	expected := map[string][]tsgo.SyntaxKind{
		specializationZeroOperation: {
			tsgo.SyntaxKindPrivateKeyword,
			tsgo.SyntaxKindStaticKeyword,
		},
		specializationHashOperation: {
			tsgo.SyntaxKindPrivateKeyword,
			tsgo.SyntaxKindStaticKeyword,
		},
		specializationEqualOperation: {
			tsgo.SyntaxKindPrivateKeyword,
			tsgo.SyntaxKindStaticKeyword,
		},
		specializationCopyOperation: {
			tsgo.SyntaxKindPrivateKeyword,
			tsgo.SyntaxKindStaticKeyword,
		},
		specializationCopyValueOperation: {
			tsgo.SyntaxKindPrivateKeyword,
			tsgo.SyntaxKindStaticKeyword,
		},
		specializationFindOperation: {
			tsgo.SyntaxKindPrivateKeyword,
		},
		names.nilMember:    {tsgo.SyntaxKindStaticKeyword},
		names.makeMember:   {tsgo.SyntaxKindStaticKeyword},
		names.lookup:       nil,
		names.lookupOK:     nil,
		names.store:        nil,
		names.deleteMember: nil,
		names.length:       nil,
		names.isNil:        nil,
	}
	if capabilities.Clear {
		expected[names.clear] = nil
	}
	if len(members) != len(expected)+1 {
		return specializationShapeError(
			role,
			"member count",
			fmt.Sprintf("%d", len(members)),
		)
	}
	constructors := 0
	seen := make(map[string]bool, len(expected))
	for _, member := range members {
		switch member := member.(type) {
		case tsgo.ConstructorDeclaration:
			constructors++
			if !specializationConstructor(member) {
				return specializationShapeError(
					role,
					"constructor",
					fmt.Sprintf("%d", len(member.Parameters())),
				)
			}
		case tsgo.MethodDeclaration:
			name, ok := specializationName(member.Name())
			modifiers, expectedMethod := expected[name]
			if !ok ||
				!expectedMethod ||
				seen[name] ||
				!specializationModifiers(member.Modifiers(), modifiers) {
				return specializationShapeError(role, "method", name)
			}
			seen[name] = true
		default:
			return specializationShapeError(
				role,
				"unexpected stored member",
				fmt.Sprintf("%T", member),
			)
		}
	}
	if constructors != 1 || len(seen) != len(expected) {
		return specializationShapeError(
			role,
			"definition set",
			fmt.Sprintf("constructors=%d methods=%d", constructors, len(seen)),
		)
	}
	return nil
}

func specializationConstructor(
	source tsgo.ConstructorDeclaration,
) bool {
	parameters := source.Parameters()
	if len(parameters) != 3 {
		return false
	}
	for index, expected := range []string{"zeroValue", "buckets", "count"} {
		name, ok := specializationName(parameters[index].Name())
		if !ok || name != expected {
			return false
		}
	}
	return true
}

func specializationModifiers(
	actual []tsgo.ModifierLike,
	expected []tsgo.SyntaxKind,
) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index, modifier := range actual {
		if modifier.Kind() != expected[index] {
			return false
		}
	}
	return true
}

func specializationName(source tsgo.Node) (string, bool) {
	identifier, ok := source.(tsgo.Identifier)
	return identifier.Text(), ok
}

func specializationShapeError(
	role api.Role,
	part string,
	value string,
) error {
	return &api.InvariantError{
		Role: role,
		Reason: fmt.Sprintf(
			"aggregate map specialization %s is invalid: %s",
			part,
			value,
		),
	}
}
