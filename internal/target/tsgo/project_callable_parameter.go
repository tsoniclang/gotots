package tsgo

import (
	"fmt"
	"slices"
)

type ProjectPrimitiveKind uint8

const (
	ProjectPrimitiveInvalid ProjectPrimitiveKind = iota
	ProjectPrimitiveString
	ProjectPrimitiveNumber
	ProjectPrimitiveBoolean
	ProjectPrimitiveBigInt
)

const (
	typeFlagString  uint32 = 1 << 5
	typeFlagNumber  uint32 = 1 << 6
	typeFlagBigInt  uint32 = 1 << 7
	typeFlagBoolean uint32 = 1 << 8
)

type ProjectPrimitiveParameter struct {
	kind     ProjectPrimitiveKind
	optional bool
}

func (p ProjectPrimitiveParameter) Kind() ProjectPrimitiveKind { return p.kind }
func (p ProjectPrimitiveParameter) Optional() bool             { return p.optional }

func (p *ProjectInspection) CallableParameterPrimitive(
	target projectCallable,
	parameter int,
) (ProjectPrimitiveParameter, bool, error) {
	if p == nil || target == nil {
		return ProjectPrimitiveParameter{}, false, &ProjectInspectionError{
			Operation: "callable parameter primitive",
			Reason:    "target is absent",
		}
	}
	signature, err := p.singleCallSignature(target, target.callableSubject())
	if err != nil {
		return ProjectPrimitiveParameter{}, false, err
	}
	if parameter < 0 || parameter >= len(signature.Parameters) {
		return ProjectPrimitiveParameter{}, false, &ProjectInspectionError{
			Operation: "callable parameter primitive",
			Reason:    "parameter is outside the callable signature",
		}
	}
	parameterType, err := p.projectSymbolType(
		signature.Parameters[parameter],
		"callable parameter primitive",
	)
	if err != nil {
		return ProjectPrimitiveParameter{}, false, err
	}
	optional := false
	if parameterType.Flags&typeFlagUnion != 0 {
		members, err := p.compositeTypes(parameterType.ID)
		if err != nil {
			return ProjectPrimitiveParameter{}, false, err
		}
		var value *typeResponse
		for index := range members {
			member := &members[index]
			if member.Flags&typeFlagUndefined != 0 {
				if optional {
					return ProjectPrimitiveParameter{}, false, nil
				}
				optional = true
				continue
			}
			if value != nil {
				return ProjectPrimitiveParameter{}, false, nil
			}
			value = member
		}
		if !optional || value == nil {
			return ProjectPrimitiveParameter{}, false, nil
		}
		parameterType = *value
	}
	kind := projectPrimitiveKind(parameterType.Flags)
	if kind == ProjectPrimitiveInvalid {
		return ProjectPrimitiveParameter{}, false, nil
	}
	return ProjectPrimitiveParameter{kind: kind, optional: optional}, true, nil
}

func projectPrimitiveKind(flags uint32) ProjectPrimitiveKind {
	switch {
	case flags&typeFlagString != 0:
		return ProjectPrimitiveString
	case flags&typeFlagNumber != 0:
		return ProjectPrimitiveNumber
	case flags&typeFlagBoolean != 0:
		return ProjectPrimitiveBoolean
	case flags&typeFlagBigInt != 0:
		return ProjectPrimitiveBigInt
	default:
		return ProjectPrimitiveInvalid
	}
}

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

func (p *ProjectInspection) CallableParameterTypeString(
	target projectCallable,
	parameter int,
) (string, error) {
	if p == nil || target == nil {
		return "", &ProjectInspectionError{
			Operation: "callable parameter type",
			Reason:    "target is absent",
		}
	}
	signature, err := p.singleCallSignature(target, target.callableSubject())
	if err != nil {
		return "", err
	}
	if parameter < 0 || parameter >= len(signature.Parameters) {
		return "", &ProjectInspectionError{
			Operation: "callable parameter type",
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
		"callable parameter type",
	)
	if err != nil {
		return "", err
	}
	return p.projectTypeString(parameterType.ID, "callable parameter type")
}

func (p *ProjectInspection) CallableReturnTypeString(
	target projectCallable,
) (string, error) {
	if p == nil || target == nil {
		return "", &ProjectInspectionError{
			Operation: "callable return type",
			Reason:    "target is absent",
		}
	}
	signature, err := p.singleCallSignature(target, target.callableSubject())
	if err != nil {
		return "", err
	}
	result, err := p.signatureReturn(signature.ID, target.callableSubject())
	if err != nil {
		return "", err
	}
	return p.projectTypeString(result.ID, "callable return type")
}

func (p *ProjectInspection) CallableReturnTypeIdentity(
	target projectCallable,
) (ProjectTypeIdentity, error) {
	if p == nil || target == nil {
		return ProjectTypeIdentity{}, &ProjectInspectionError{
			Operation: "callable return type identity",
			Reason:    "target is absent",
		}
	}
	signature, err := p.singleCallSignature(target, target.callableSubject())
	if err != nil {
		return ProjectTypeIdentity{}, err
	}
	result, err := p.signatureReturn(signature.ID, target.callableSubject())
	if err != nil {
		return ProjectTypeIdentity{}, err
	}
	return p.projectTypeIdentity(result.ID)
}

func (p *ProjectInspection) projectTypeString(
	typeID uint32,
	operation string,
) (string, error) {
	if typeID == 0 {
		return "", &ProjectInspectionError{
			Operation: operation,
			Reason:    "type is absent",
		}
	}
	var result string
	if err := requestProjectJSON(
		p.client,
		"typeToString",
		typeToStringParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			Type:     typeID,
		},
		&result,
	); err != nil {
		return "", err
	}
	if result == "" {
		return "", &ProjectInspectionError{
			Operation: operation,
			Reason:    "type string is absent",
		}
	}
	return result, nil
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
