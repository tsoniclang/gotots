package tsgo

import "fmt"

const typeFlagUnionOrIntersection uint32 = 1<<27 | 1<<28

type CallableEffect uint8

const (
	CallableEffectInvalid CallableEffect = iota
	CallableEffectSynchronous
	CallableEffectAsynchronous
	CallableEffectAwaitable
)

func (e CallableEffect) Valid() bool {
	return e == CallableEffectSynchronous ||
		e == CallableEffectAsynchronous ||
		e == CallableEffectAwaitable
}

type projectCallable interface {
	callableTypeIDs() []uint32
	callableSubject() string
}

type projectCallableType struct {
	typeID  uint32
	subject string
}

func (t projectCallableType) callableTypeIDs() []uint32 {
	return []uint32{t.typeID}
}

func (t projectCallableType) callableSubject() string {
	return t.subject
}

func (e ProjectExport) callableTypeIDs() []uint32 {
	return []uint32{e.declaredTypeID, e.typeID}
}

func (e ProjectExport) callableSubject() string {
	return e.name
}

func (m ProjectMember) callableTypeIDs() []uint32 {
	return []uint32{m.typeID}
}

func (m ProjectMember) callableSubject() string {
	return m.name
}

func (p *ProjectInspection) CallableEffect(
	target projectCallable,
	asyncMarker ProjectExport,
) (CallableEffect, error) {
	if p == nil || target == nil {
		return CallableEffectInvalid, &ProjectInspectionError{
			Operation: "callable effect",
			Reason:    "target is absent",
		}
	}
	targetSignature, err := p.singleCallSignature(target, target.callableSubject())
	if err != nil {
		return CallableEffectInvalid, err
	}
	markerSignature, err := p.singleCallSignature(asyncMarker, "async marker")
	if err != nil {
		return CallableEffectInvalid, err
	}
	targetReturn, err := p.signatureReturn(targetSignature.ID, "target")
	if err != nil {
		return CallableEffectInvalid, err
	}
	markerReturn, err := p.signatureReturn(markerSignature.ID, "async marker")
	if err != nil {
		return CallableEffectInvalid, err
	}
	if markerReturn.Target == 0 {
		return CallableEffectInvalid, &ProjectInspectionError{
			Operation: "callable effect",
			Reason:    "async marker does not own a generic type target",
		}
	}
	if targetReturn.Flags&typeFlagUnionOrIntersection != 0 {
		members, err := p.compositeTypes(targetReturn.ID)
		if err != nil {
			return CallableEffectInvalid, err
		}
		awaitable, err := p.awaitableEffect(members, markerReturn.Target)
		if err != nil {
			return CallableEffectInvalid, err
		}
		if awaitable {
			return CallableEffectAwaitable, nil
		}
		return CallableEffectSynchronous, nil
	}
	if targetReturn.Target == markerReturn.Target {
		return CallableEffectAsynchronous, nil
	}
	return CallableEffectSynchronous, nil
}

func (p *ProjectInspection) awaitableEffect(
	members []typeResponse,
	promiseTarget uint32,
) (bool, error) {
	containsPromise := false
	for _, member := range members {
		containsPromise = containsPromise || member.Target == promiseTarget
	}
	if !containsPromise {
		return false, nil
	}
	var promised *typeResponse
	var direct []typeResponse
	for index := range members {
		member := &members[index]
		if member.Target == promiseTarget {
			if promised != nil {
				return false, &ProjectInspectionError{
					Operation: "callable effect",
					Reason:    "target return type has duplicate Promise alternatives",
				}
			}
			promised = member
			continue
		}
		direct = append(direct, *member)
	}
	if promised == nil {
		return false, nil
	}
	if len(direct) == 0 {
		return false, &ProjectInspectionError{
			Operation: "callable effect",
			Reason:    "target return type has no direct alternative",
		}
	}
	arguments, err := p.typeArguments(promised.ID)
	if err != nil {
		return false, err
	}
	if len(arguments) != 1 {
		return false, &ProjectInspectionError{
			Operation: "callable effect",
			Reason:    "target return type is not T | Promise<T>",
		}
	}
	if len(direct) == 1 && arguments[0].ID == direct[0].ID {
		return true, nil
	}
	if arguments[0].Flags&typeFlagUnionOrIntersection == 0 {
		return false, &ProjectInspectionError{
			Operation: "callable effect",
			Reason:    "target return type is not T | Promise<T>",
		}
	}
	argumentMembers, err := p.compositeTypes(arguments[0].ID)
	if err != nil {
		return false, err
	}
	if !sameTypeMembers(direct, argumentMembers) {
		return false, &ProjectInspectionError{
			Operation: "callable effect",
			Reason:    "target return type is not T | Promise<T>",
		}
	}
	return true, nil
}

func sameTypeMembers(left []typeResponse, right []typeResponse) bool {
	if len(left) != len(right) {
		return false
	}
	identities := make(map[uint32]int, len(left))
	for _, member := range left {
		identities[member.ID]++
	}
	for _, member := range right {
		if identities[member.ID] == 0 {
			return false
		}
		identities[member.ID]--
	}
	return true
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

func (p *ProjectInspection) projectSymbolType(
	symbol uint64,
	operation string,
) (typeResponse, error) {
	if symbol == 0 {
		return typeResponse{}, &ProjectInspectionError{
			Operation: operation,
			Reason:    "parameter symbol is absent",
		}
	}
	var selected *typeResponse
	if err := requestProjectJSON(
		p.client,
		"getTypeOfSymbol",
		getTypeOfSymbolParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			Symbol:   symbol,
		},
		&selected,
	); err != nil {
		return typeResponse{}, err
	}
	if selected == nil || selected.ID == 0 {
		return typeResponse{}, &ProjectInspectionError{
			Operation: operation,
			Reason:    "parameter type is absent",
		}
	}
	return *selected, nil
}

func (p *ProjectInspection) compositeTypes(source uint32) ([]typeResponse, error) {
	var selected []typeResponse
	if err := requestProjectJSON(
		p.client,
		"getTypesOfType",
		getTypePropertyParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			Type:     source,
		},
		&selected,
	); err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, &ProjectInspectionError{
			Operation: "callable effect",
			Reason:    "composite return type has no members",
		}
	}
	return selected, nil
}

func (p *ProjectInspection) singleCallSignature(
	target projectCallable,
	subject string,
) (signatureResponse, error) {
	seen := make(map[uint32]struct{})
	for _, candidate := range target.callableTypeIDs() {
		if candidate == 0 {
			continue
		}
		if _, duplicate := seen[candidate]; duplicate {
			continue
		}
		seen[candidate] = struct{}{}
		nonNullable, err := p.nonNullableType(candidate)
		if err != nil {
			return signatureResponse{}, err
		}
		var signatures []signatureResponse
		if err := requestProjectJSON(
			p.client,
			"getSignaturesOfType",
			getSignaturesOfTypeParams{
				Snapshot: p.snapshot,
				Project:  p.project,
				Type:     nonNullable,
				Kind:     0,
			},
			&signatures,
		); err != nil {
			return signatureResponse{}, err
		}
		if len(signatures) == 0 {
			continue
		}
		if len(signatures) != 1 || signatures[0].ID == 0 {
			return signatureResponse{}, &ProjectInspectionError{
				Operation: "callable effect",
				Reason: fmt.Sprintf(
					"%s has %d call signatures, want one",
					subject,
					len(signatures),
				),
			}
		}
		return signatures[0], nil
	}
	return signatureResponse{}, &ProjectInspectionError{
		Operation: "callable effect",
		Reason:    subject + " has no call signature",
	}
}

func (p *ProjectInspection) nonNullableType(source uint32) (uint32, error) {
	var selected *typeResponse
	if err := requestProjectJSON(
		p.client,
		"getNonNullableType",
		getNonNullableTypeParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			Type:     source,
		},
		&selected,
	); err != nil {
		return 0, err
	}
	if selected == nil || selected.ID == 0 {
		return 0, &ProjectInspectionError{
			Operation: "callable effect",
			Reason:    "non-nullable type is absent",
		}
	}
	return selected.ID, nil
}

func (p *ProjectInspection) signatureReturn(
	signature uint64,
	subject string,
) (typeResponse, error) {
	var selected *typeResponse
	if err := requestProjectJSON(
		p.client,
		"getReturnTypeOfSignature",
		checkerSignatureParams{
			Snapshot:  p.snapshot,
			Project:   p.project,
			Signature: signature,
		},
		&selected,
	); err != nil {
		return typeResponse{}, err
	}
	if selected == nil || selected.ID == 0 {
		return typeResponse{}, &ProjectInspectionError{
			Operation: "callable effect",
			Reason:    subject + " return type is absent",
		}
	}
	return *selected, nil
}

type getNonNullableTypeParams struct {
	Snapshot uint64 `json:"snapshot"`
	Project  string `json:"project"`
	Type     uint32 `json:"type"`
}

type getSignaturesOfTypeParams struct {
	Snapshot uint64 `json:"snapshot"`
	Project  string `json:"project"`
	Type     uint32 `json:"type"`
	Kind     int32  `json:"kind"`
}

type checkerSignatureParams struct {
	Snapshot  uint64 `json:"snapshot"`
	Project   string `json:"project"`
	Signature uint64 `json:"signature"`
}

type getTypePropertyParams struct {
	Snapshot uint64 `json:"snapshot"`
	Project  string `json:"project"`
	Type     uint32 `json:"type"`
}

type signatureResponse struct {
	ID             uint64   `json:"id"`
	TypeParameters []uint32 `json:"typeParameters,omitempty"`
	Parameters     []uint64 `json:"parameters,omitempty"`
}
