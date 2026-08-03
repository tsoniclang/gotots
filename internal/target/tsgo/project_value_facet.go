package tsgo

import "fmt"

const (
	typeAliasNameBit           = uint32(1 << 1)
	typeAliasTypeParametersBit = uint32(1 << 2)
	typeAliasTypeBit           = uint32(1 << 3)
	typeParameterNameBit       = uint32(1 << 1)
	typeParameterConstraintBit = uint32(1 << 2)
	typeParameterExpressionBit = uint32(1 << 3)
	typeParameterDefaultBit    = uint32(1 << 4)
	typeReferenceNameBit       = uint32(1 << 0)
)

type projectTypeCallable struct {
	typeID  uint32
	subject string
}

func (c projectTypeCallable) callableTypeIDs() []uint32 {
	return []uint32{c.typeID}
}

func (c projectTypeCallable) callableSubject() string {
	return c.subject
}

func (p *ProjectInspection) CallableValueFacetEffect(
	target ProjectExport,
	asyncMarker ProjectExport,
) (CallableEffect, error) {
	defaultType, err := p.callableValueFacetDefault(target)
	if err != nil {
		return CallableEffectInvalid, err
	}
	return p.CallableEffect(
		projectTypeCallable{
			typeID:  defaultType,
			subject: target.name + " default ABI",
		},
		asyncMarker,
	)
}

func (p *ProjectInspection) callableValueFacetDefault(
	target ProjectExport,
) (uint32, error) {
	if p == nil || p.client == nil {
		return 0, valueFacetError(target.name, "project inspection is invalid")
	}
	declarationHandles, err := p.valueFacetDeclarationHandles(target)
	if err != nil {
		return 0, err
	}
	if len(declarationHandles) != 1 {
		return 0, valueFacetError(
			target.name,
			fmt.Sprintf(
				"facet has %d type-alias declarations across %d export and %d resolved declarations, want one",
				len(declarationHandles),
				len(target.exportHandles),
				len(target.handles),
			),
		)
	}
	declaration, kind, sourcePath, err := parseProjectNodeHandle(
		declarationHandles[0],
	)
	if err != nil {
		return 0, valueFacetError(target.name, err.Error())
	}
	if kind != uint32(SyntaxKindTypeAliasDeclaration) {
		return 0, valueFacetError(target.name, "facet is not a type alias")
	}
	source, err := p.projectSourceEvidence(sourcePath)
	if err != nil {
		return 0, err
	}
	alias, ok := source.node(declaration, kind)
	if !ok {
		return 0, valueFacetError(
			target.name,
			"declaration does not identify its official AST node",
		)
	}
	requiredAlias := typeAliasNameBit |
		typeAliasTypeParametersBit |
		typeAliasTypeBit
	if alias.data&requiredAlias != requiredAlias {
		return 0, valueFacetError(target.name, "facet alias shape is incomplete")
	}
	typeParameter, aliasType, err := source.valueFacetAliasChildren(declaration)
	if err != nil {
		return 0, valueFacetError(target.name, err.Error())
	}
	parameter, ok := source.node(typeParameter, uint32(SyntaxKindTypeParameter))
	if !ok {
		return 0, valueFacetError(target.name, "facet type parameter is invalid")
	}
	exactParameter := typeParameterNameBit | typeParameterDefaultBit
	if parameter.data != exactParameter ||
		parameter.data&typeParameterConstraintBit != 0 ||
		parameter.data&typeParameterExpressionBit != 0 {
		return 0, valueFacetError(
			target.name,
			"facet type parameter must be unconstrained with one default",
		)
	}
	parameterName, defaultNode, err := source.valueFacetParameterChildren(
		typeParameter,
	)
	if err != nil {
		return 0, valueFacetError(target.name, err.Error())
	}
	aliasTypeNode := source.nodes[aliasType]
	if aliasTypeNode.kind != uint32(SyntaxKindTypeReference) ||
		aliasTypeNode.data != typeReferenceNameBit {
		return 0, valueFacetError(
			target.name,
			"facet alias body is not the direct type parameter",
		)
	}
	parameterType, err := p.typeAtProjectNode(
		sourcePath,
		parameterName,
		"facet type parameter",
	)
	if err != nil {
		return 0, err
	}
	aliasValueType, err := p.typeFromProjectTypeNode(
		sourcePath,
		aliasType,
		"facet alias body",
	)
	if err != nil {
		return 0, err
	}
	if parameterType.ID != aliasValueType.ID {
		return 0, valueFacetError(
			target.name,
			"facet alias body does not resolve to its type parameter",
		)
	}
	defaultType, err := p.typeFromProjectTypeNode(
		sourcePath,
		defaultNode,
		"facet default ABI",
	)
	if err != nil {
		return 0, err
	}
	return defaultType.ID, nil
}

func (p *ProjectInspection) valueFacetDeclarationHandles(
	target ProjectExport,
) ([]string, error) {
	seen := make(map[string]struct{})
	result := make(
		[]string,
		0,
		len(target.exportHandles)+len(target.handles),
	)
	for _, handles := range [][]string{
		target.exportHandles,
		target.handles,
	} {
		for _, handle := range handles {
			index, kind, sourcePath, err := parseProjectNodeHandle(handle)
			if err != nil {
				return nil, valueFacetError(target.name, err.Error())
			}
			if kind == uint32(SyntaxKindTypeParameter) {
				source, sourceErr := p.projectSourceEvidence(sourcePath)
				if sourceErr != nil {
					return nil, sourceErr
				}
				index, kind = source.enclosingTypeAlias(index)
				if index != 0 {
					handle = fmt.Sprintf("%d.%d.%s", index, kind, sourcePath)
				}
			}
			if kind != uint32(SyntaxKindTypeAliasDeclaration) {
				continue
			}
			if _, duplicate := seen[handle]; duplicate {
				continue
			}
			seen[handle] = struct{}{}
			result = append(result, handle)
		}
	}
	return result, nil
}

func (s projectSourceEvidence) enclosingTypeAlias(
	index uint32,
) (uint32, uint32) {
	for index != 0 && index < uint32(len(s.nodes)) {
		node := s.nodes[index]
		if node.kind == uint32(SyntaxKindTypeAliasDeclaration) {
			return index, node.kind
		}
		index = node.parent
	}
	return 0, 0
}

func (s projectSourceEvidence) valueFacetAliasChildren(
	declaration uint32,
) (uint32, uint32, error) {
	children := s.directChildren(declaration)
	if len(children) < 3 {
		return 0, 0, fmt.Errorf("facet alias children are incomplete")
	}
	var typeParameter uint32
	var aliasType uint32
	for _, child := range children {
		node := s.nodes[child]
		if node.kind == kindNodeList {
			for _, listed := range s.directChildren(child) {
				if s.nodes[listed].kind != uint32(SyntaxKindTypeParameter) {
					continue
				}
				if typeParameter != 0 {
					return 0, 0, fmt.Errorf("facet has multiple type parameters")
				}
				typeParameter = listed
			}
			continue
		}
		if node.kind == uint32(SyntaxKindIdentifier) && aliasType == 0 {
			continue
		}
		if typeParameter != 0 && aliasType == 0 {
			aliasType = child
		}
	}
	if typeParameter == 0 || aliasType == 0 {
		return 0, 0, fmt.Errorf("facet type parameter or alias body is absent")
	}
	return typeParameter, aliasType, nil
}

func (s projectSourceEvidence) valueFacetParameterChildren(
	parameter uint32,
) (uint32, uint32, error) {
	children := s.directChildren(parameter)
	if len(children) < 2 ||
		s.nodes[children[0]].kind != uint32(SyntaxKindIdentifier) {
		return 0, 0, fmt.Errorf("facet type parameter children are invalid")
	}
	return children[0], children[1], nil
}

func valueFacetError(subject string, reason string) error {
	return &ProjectInspectionError{
		Operation: "callable value facet",
		Path:      subject,
		Reason:    reason,
	}
}
