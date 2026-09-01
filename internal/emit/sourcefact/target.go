package sourcefact

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	attribute "github.com/tsoniclang/gotots/internal/emit/marker/attribute"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Target struct {
	typeNode   tsgo.TypeNode
	memberKind attribute.MemberKind
	member     string
}

func NewTarget(typeNode tsgo.TypeNode) (Target, error) {
	if typeNode == nil {
		return Target{}, &Error{Reason: "source-fact target type is nil"}
	}
	return Target{typeNode: typeNode}, nil
}

func NewMemberTarget(
	typeNode tsgo.TypeNode,
	kind attribute.MemberKind,
	member string,
) (Target, error) {
	if typeNode == nil || member == "" || kind == attribute.MemberInvalid {
		return Target{}, &Error{Reason: "source-fact member target is invalid"}
	}
	return Target{
		typeNode:   typeNode,
		memberKind: kind,
		member:     member,
	}, nil
}

func (t Target) apply(
	context api.Context,
	fact api.RuntimeSymbol,
	arguments ...tsgo.Expression,
) (api.StatementEmission, error) {
	if t.typeNode == nil {
		return api.StatementEmission{}, &Error{Reason: "source-fact target is invalid"}
	}
	if t.memberKind == attribute.MemberInvalid {
		if t.member != "" {
			return api.StatementEmission{}, &Error{Reason: "source-fact target member is inconsistent"}
		}
		return attribute.Apply(context, t.typeNode, fact, arguments...)
	}
	return attribute.ApplyMember(
		context,
		t.typeNode,
		t.memberKind,
		t.member,
		fact,
		arguments...,
	)
}
