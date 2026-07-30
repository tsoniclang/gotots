package api

import "go/types"

func MethodReceiverTypeName(method *types.Func) *types.TypeName {
	if method == nil {
		return nil
	}
	method = method.Origin()
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return nil
	}
	receiver := types.Unalias(signature.Recv().Type())
	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = types.Unalias(pointer.Elem())
	}
	named, ok := receiver.(*types.Named)
	if !ok || named.Obj() == nil {
		return nil
	}
	return named.Origin().Obj()
}

func ValueReceiverTypeName(method *types.Func) *types.TypeName {
	if method == nil {
		return nil
	}
	method = method.Origin()
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return nil
	}
	if _, pointer := types.Unalias(signature.Recv().Type()).(*types.Pointer); pointer {
		return nil
	}
	return MethodReceiverTypeName(method)
}

func NewClassMethodRequirement(
	owner *types.TypeName,
	method *types.Func,
) (DeclarationRequirement, error) {
	if owner == nil ||
		method == nil ||
		method.Origin() != method ||
		method.Name() == "_" ||
		MethodReceiverTypeName(method) != owner {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "class-method requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:       MustSourceArtifactOwner(owner),
		kind:        DeclarationRequirementClassMethod,
		typeName:    owner,
		classMethod: method,
	}, nil
}

func NewClassMethodRequest(
	owner *types.TypeName,
	method *types.Func,
) (RootRequest, error) {
	requirement, err := NewClassMethodRequirement(owner, method)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r DeclarationRequirement) ClassMethod() (
	*types.TypeName,
	*types.Func,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementClassMethod {
		return nil, nil, false
	}
	return r.typeName, r.classMethod, true
}

func NewValueReceiverCopyRequirement(
	method *types.Func,
) (DeclarationRequirement, error) {
	if method == nil ||
		method.Origin() != method ||
		ValueReceiverTypeName(method) == nil {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "value-receiver copy requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner: MustSourceArtifactOwner(method),
		kind:  DeclarationRequirementValueReceiverCopy,
	}, nil
}

func NewValueReceiverCopyRequest(
	method *types.Func,
) (RootRequest, error) {
	requirement, err := NewValueReceiverCopyRequirement(method)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r DeclarationRequirement) ValueReceiverCopy() (
	*types.Func,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementValueReceiverCopy {
		return nil, false
	}
	method, ok := r.owner.Source()
	selected, methodOK := method.(*types.Func)
	return selected, ok && methodOK
}
