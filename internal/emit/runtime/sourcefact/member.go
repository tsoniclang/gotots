package sourcefact

import "github.com/tsoniclang/gotots/internal/target/tsgo"

type declarationMember struct {
	name   string
	kind   string
	static bool
}

func (m declarationMember) targetName() string {
	if m.static {
		return "static"
	}
	return "instance"
}

func declarationMembers(
	declaration tsgo.Statement,
) ([]declarationMember, error) {
	var selected []declarationMember
	switch owner := declaration.(type) {
	case tsgo.ClassDeclaration:
		selected = classMembers(owner.Members())
	case tsgo.InterfaceDeclaration:
		selected = interfaceMembers(owner.Members())
	default:
		return nil, nil
	}
	result := make([]declarationMember, 0, len(selected))
	seen := make(map[declarationMember]struct{}, len(selected))
	for _, member := range selected {
		if member.name == "" {
			continue
		}
		if _, duplicate := seen[member]; duplicate {
			continue
		}
		seen[member] = struct{}{}
		result = append(result, member)
	}
	return result, nil
}

func classMembers(elements []tsgo.ClassElement) []declarationMember {
	result := make([]declarationMember, 0, len(elements))
	for _, element := range elements {
		switch selected := element.(type) {
		case tsgo.MethodDeclaration:
			result = appendNamedClassMember(
				result,
				selected.Name(),
				selected.Modifiers(),
				"method",
			)
		case tsgo.PropertyDeclaration:
			result = appendNamedClassMember(
				result,
				selected.Name(),
				selected.Modifiers(),
				"property",
			)
		case tsgo.GetAccessorDeclaration:
			result = appendNamedClassMember(
				result,
				selected.Name(),
				selected.Modifiers(),
				"property",
			)
		case tsgo.SetAccessorDeclaration:
			result = appendNamedClassMember(
				result,
				selected.Name(),
				selected.Modifiers(),
				"property",
			)
		case tsgo.ConstructorDeclaration:
			for _, parameter := range selected.Parameters() {
				result = appendParameterProperty(result, parameter)
			}
		}
	}
	return result
}

func interfaceMembers(elements []tsgo.TypeElement) []declarationMember {
	result := make([]declarationMember, 0, len(elements))
	for _, element := range elements {
		switch selected := element.(type) {
		case tsgo.MethodSignatureDeclaration:
			result = appendNamedMember(result, selected.Name(), "method", false)
		case tsgo.PropertySignatureDeclaration:
			result = appendNamedMember(result, selected.Name(), "property", false)
		}
	}
	return result
}

func appendNamedClassMember(
	result []declarationMember,
	name tsgo.PropertyName,
	modifiers []tsgo.ModifierLike,
	kind string,
) []declarationMember {
	if hasModifier(modifiers, tsgo.SyntaxKindPrivateKeyword) ||
		hasModifier(modifiers, tsgo.SyntaxKindProtectedKeyword) {
		return result
	}
	return appendNamedMember(
		result,
		name,
		kind,
		hasModifier(modifiers, tsgo.SyntaxKindStaticKeyword),
	)
}

func appendParameterProperty(
	result []declarationMember,
	parameter tsgo.ParameterDeclaration,
) []declarationMember {
	modifiers := parameter.Modifiers()
	if hasModifier(modifiers, tsgo.SyntaxKindPrivateKeyword) ||
		hasModifier(modifiers, tsgo.SyntaxKindProtectedKeyword) {
		return result
	}
	if !hasModifier(modifiers, tsgo.SyntaxKindPublicKeyword) &&
		!hasModifier(modifiers, tsgo.SyntaxKindReadonlyKeyword) {
		return result
	}
	identifier, ok := parameter.Name().(tsgo.Identifier)
	if !ok {
		return result
	}
	return append(result, declarationMember{
		name: identifier.Text(),
		kind: "property",
	})
}

func appendNamedMember(
	result []declarationMember,
	name tsgo.PropertyName,
	kind string,
	static bool,
) []declarationMember {
	identifier, ok := name.(tsgo.Identifier)
	if !ok {
		return result
	}
	return append(result, declarationMember{
		name:   identifier.Text(),
		kind:   kind,
		static: static,
	})
}

func hasModifier(modifiers []tsgo.ModifierLike, kind tsgo.SyntaxKind) bool {
	for _, modifier := range modifiers {
		if modifier.Kind() == kind {
			return true
		}
	}
	return false
}
