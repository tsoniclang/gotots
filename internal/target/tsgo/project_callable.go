package tsgo

import "fmt"

const typeFlagUnionOrIntersection uint32 = 1<<27 | 1<<28

type CallableEffect uint8

const (
	CallableEffectInvalid CallableEffect = iota
	CallableEffectSynchronous
	CallableEffectAsynchronous
)

func (e CallableEffect) Valid() bool {
	return e == CallableEffectSynchronous || e == CallableEffectAsynchronous
}

type projectCallable interface {
	callableTypeIDs() []uint32
	callableSubject() string
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
	targetReturn, err := p.signatureReturn(targetSignature, "target")
	if err != nil {
		return CallableEffectInvalid, err
	}
	markerReturn, err := p.signatureReturn(markerSignature, "async marker")
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
		for _, member := range members {
			if member.Target == markerReturn.Target {
				return CallableEffectInvalid, &ProjectInspectionError{
					Operation: "callable effect",
					Reason:    "target return type has multiple effect alternatives",
				}
			}
		}
		return CallableEffectSynchronous, nil
	}
	if targetReturn.Target == markerReturn.Target {
		return CallableEffectAsynchronous, nil
	}
	return CallableEffectSynchronous, nil
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
) (uint64, error) {
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
			return 0, err
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
			return 0, err
		}
		if len(signatures) == 0 {
			continue
		}
		if len(signatures) != 1 || signatures[0].ID == 0 {
			return 0, &ProjectInspectionError{
				Operation: "callable effect",
				Reason: fmt.Sprintf(
					"%s has %d call signatures, want one",
					subject,
					len(signatures),
				),
			}
		}
		return signatures[0].ID, nil
	}
	return 0, &ProjectInspectionError{
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
	ID uint64 `json:"id"`
}
