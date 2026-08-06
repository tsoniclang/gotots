package tsgo

import (
	"fmt"
	"slices"
)

func (p *ProjectInspection) CallableTypeParameterCount(
	target projectCallable,
) (int, error) {
	if p == nil || target == nil {
		return 0, &ProjectInspectionError{
			Operation: "callable type parameters",
			Reason:    "target is absent",
		}
	}
	signature, err := p.singleCallSignature(
		target,
		target.callableSubject(),
	)
	if err != nil {
		return 0, err
	}
	return len(signature.TypeParameters), nil
}

func (p *ProjectInspection) CallableParameterCount(
	target projectCallable,
) (int, error) {
	if p == nil || target == nil {
		return 0, &ProjectInspectionError{
			Operation: "callable parameters",
			Reason:    "target is absent",
		}
	}
	signature, err := p.singleCallSignature(
		target,
		target.callableSubject(),
	)
	if err != nil {
		return 0, err
	}
	return len(signature.Parameters), nil
}

func (p *ProjectInspection) CallableParameterEffect(
	target projectCallable,
	parameter int,
	asyncMarker ProjectExport,
) (CallableEffect, error) {
	if p == nil || target == nil {
		return CallableEffectInvalid, &ProjectInspectionError{
			Operation: "callable parameter effect",
			Reason:    "target is absent",
		}
	}
	signature, err := p.singleCallSignature(target, target.callableSubject())
	if err != nil {
		return CallableEffectInvalid, err
	}
	if parameter < 0 || parameter >= len(signature.Parameters) {
		return CallableEffectInvalid, &ProjectInspectionError{
			Operation: "callable parameter effect",
			Reason: fmt.Sprintf(
				"%s parameter %d is outside %d parameters",
				target.callableSubject(),
				parameter,
				len(signature.Parameters),
			),
		}
	}
	parameterType, err := p.projectSymbolType(
		signature.Parameters[parameter],
		"callable parameter effect",
	)
	if err != nil {
		return CallableEffectInvalid, err
	}
	return p.CallableEffect(projectCallableType{
		typeID: parameterType.ID,
		subject: fmt.Sprintf(
			"%s parameter %d",
			target.callableSubject(),
			parameter,
		),
	}, asyncMarker)
}

func (p *ProjectInspection) CallableParameterTypeIdentity(
	target projectCallable,
	parameter int,
) (ProjectTypeIdentity, error) {
	if p == nil || target == nil {
		return ProjectTypeIdentity{}, &ProjectInspectionError{
			Operation: "callable parameter type identity",
			Reason:    "target is absent",
		}
	}
	signature, err := p.singleCallSignature(target, target.callableSubject())
	if err != nil {
		return ProjectTypeIdentity{}, err
	}
	if parameter < 0 || parameter >= len(signature.Parameters) {
		return ProjectTypeIdentity{}, &ProjectInspectionError{
			Operation: "callable parameter type identity",
			Reason: fmt.Sprintf(
				"%s parameter %d is outside %d parameters",
				target.callableSubject(),
				parameter,
				len(signature.Parameters),
			),
		}
	}
	parameterType, err := p.projectSymbolType(
		signature.Parameters[parameter],
		"callable parameter type identity",
	)
	if err != nil {
		return ProjectTypeIdentity{}, err
	}
	return p.projectTypeIdentity(parameterType.ID)
}

func (p *ProjectInspection) CallableParameterTypeArguments(
	target projectCallable,
	parameter int,
) ([]ProjectTypeIdentity, error) {
	if p == nil || target == nil {
		return nil, &ProjectInspectionError{
			Operation: "callable parameter type arguments",
			Reason:    "target is absent",
		}
	}
	signature, err := p.singleCallSignature(target, target.callableSubject())
	if err != nil {
		return nil, err
	}
	if parameter < 0 || parameter >= len(signature.Parameters) {
		return nil, &ProjectInspectionError{
			Operation: "callable parameter type arguments",
			Reason: fmt.Sprintf(
				"%s parameter %d is outside %d parameters",
				target.callableSubject(),
				parameter,
				len(signature.Parameters),
			),
		}
	}
	parameterType, err := p.projectSymbolType(
		signature.Parameters[parameter],
		"callable parameter type arguments",
	)
	if err != nil {
		return nil, err
	}
	arguments, err := p.typeArguments(parameterType.ID)
	if err != nil {
		return nil, err
	}
	result := make([]ProjectTypeIdentity, len(arguments))
	for index, argument := range arguments {
		result[index], err = p.projectTypeIdentity(argument.ID)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (p *ProjectInspection) CallableParameterNames(
	target projectDeclaredCallable,
) ([]string, error) {
	if p == nil || target == nil {
		return nil, &ProjectInspectionError{
			Operation: "callable parameter names",
			Reason:    "target is absent",
		}
	}
	handles := target.scalarDeclarationHandles()
	if len(handles) == 0 {
		return nil, &ProjectInspectionError{
			Operation: "callable parameter names",
			Path:      target.callableSubject(),
			Reason:    "declaration is absent",
		}
	}
	var result []string
	for index, handle := range handles {
		names, err := p.callableDeclarationParameterNames(handle)
		if err != nil {
			return nil, err
		}
		if index == 0 {
			result = names
			continue
		}
		if !slices.Equal(result, names) {
			return nil, &ProjectInspectionError{
				Operation: "callable parameter names",
				Path:      target.callableSubject(),
				Reason:    "declarations disagree on parameter names",
			}
		}
	}
	return slices.Clone(result), nil
}

func (p *ProjectInspection) callableDeclarationParameterNames(
	handle string,
) ([]string, error) {
	sourcePath, source, parameters, err := p.callableDeclarationParameters(handle)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(parameters))
	for parameterIndex, parameter := range parameters {
		var nameNode uint32
		for _, child := range source.directChildren(parameter) {
			if source.nodes[child].kind != uint32(SyntaxKindIdentifier) {
				continue
			}
			if nameNode != 0 {
				return nil, projectNodeEvidenceError(
					fmt.Sprintf("callable parameter %d", parameterIndex),
					"identifier name is duplicated",
				)
			}
			nameNode = child
		}
		if nameNode == 0 {
			return nil, projectNodeEvidenceError(
				fmt.Sprintf("callable parameter %d", parameterIndex),
				"identifier name is absent",
			)
		}
		symbol, err := p.symbolAtProjectNode(
			sourcePath,
			nameNode,
			fmt.Sprintf("callable parameter %d", parameterIndex),
		)
		if err != nil {
			return nil, err
		}
		names[parameterIndex] = symbol.Name
	}
	return names, nil
}

func (p *ProjectInspection) callableDeclarationParameters(
	handle string,
) (string, projectSourceEvidence, []uint32, error) {
	index, kind, sourcePath, err := parseProjectNodeHandle(handle)
	if err != nil {
		return "", projectSourceEvidence{}, nil, err
	}
	source, err := p.projectSourceEvidence(sourcePath)
	if err != nil {
		return "", projectSourceEvidence{}, nil, err
	}
	if _, ok := source.node(index, kind); !ok {
		return "", projectSourceEvidence{}, nil, projectNodeEvidenceError(
			"callable parameters",
			"declaration handle is invalid",
		)
	}
	var parameters []uint32
	for _, child := range source.directChildren(index) {
		if source.nodes[child].kind != kindNodeList {
			continue
		}
		for _, item := range source.directChildren(child) {
			if source.nodes[item].kind == uint32(SyntaxKindParameter) {
				parameters = append(parameters, item)
			}
		}
	}
	return sourcePath, source, parameters, nil
}
