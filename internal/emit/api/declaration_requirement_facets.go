package api

import (
	"go/ast"
	"go/token"
	"go/types"
)

func NewCooperativeCallableRequirement(
	facet CallableFacet,
) (DeclarationRequirement, error) {
	if !facet.Valid() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "cooperative callable facet is invalid",
		}
	}
	return DeclarationRequirement{
		owner:         facet.Owner(),
		kind:          DeclarationRequirementCooperativeCallable,
		callableFacet: facet,
	}, nil
}

func NewCooperativeCallableRequest(
	facet CallableFacet,
) (RootRequest, error) {
	requirement, err := NewCooperativeCallableRequirement(facet)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewCallableABIRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactCallableABI,
		DeclarationRequirementCallableABI,
		"callable ABI",
	)
}

func NewCallableABIRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewCallableABIRequirement(artifact)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r DeclarationRequirement) CooperativeCallable() (
	CallableFacet,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementCooperativeCallable {
		return CallableFacet{}, false
	}
	return r.callableFacet, true
}

func (r DeclarationRequirement) CallableABI() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementCallableABI,
		GeneratedArtifactCallableABI,
	)
}

func (r DeclarationRequirement) validCooperativeCallable() bool {
	if !r.owner.Valid() ||
		r.operation != NamedStructOperationInvalid ||
		r.typeName != nil ||
		r.variable != nil ||
		r.constant != nil ||
		r.projection != types.Invalid ||
		r.generated != nil ||
		r.anonymousDemand != AnonymousStructDemandInvalid ||
		r.mapDemand != MapSpecializationDemandInvalid ||
		r.genericOperation != nil ||
		!r.callableFacet.Valid() {
		return false
	}
	return r.owner == r.callableFacet.Owner()
}

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

// ConstantProjectionType resolves a validated concrete constant-capable basic
// representation. Raw types.BasicKind values are representable outside this
// package, so every public projection boundary uses this owner before indexing
// types.Typ. Untyped kinds and unsafe.Pointer are not projection types.
func ConstantProjectionType(
	projection types.BasicKind,
) (*types.Basic, bool) {
	index := int(projection)
	if projection == types.Invalid ||
		index < 0 ||
		index >= len(types.Typ) {
		return nil, false
	}
	selected := types.Typ[index]
	if selected == nil ||
		selected.Info()&types.IsUntyped != 0 ||
		selected.Info()&(types.IsBoolean|
			types.IsInteger|
			types.IsFloat|
			types.IsComplex|
			types.IsString) == 0 {
		return nil, false
	}
	return selected, true
}

// ConstantProjectionName is the exported name of an untyped constant's
// projection at one target basic representation. The `$` separator cannot occur
// in a Go source identifier, so a projection name never collides with a user
// declaration, and distinct (constant, representation) pairs never collide with
// each other. Both the declaration owner and every use site derive the name
// through this one function.
func ConstantProjectionName(
	base string,
	projection types.BasicKind,
) (string, error) {
	selected, ok := ConstantProjectionType(projection)
	if base == "" || !ok {
		return "", &NameError{
			Name:   base,
			Reason: "constant projection identity is invalid",
		}
	}
	return base + "$" + selected.Name(), nil
}

// NewConstantProjectionRequirement requires one untyped constant to be
// projected once at the given target basic representation.
func NewConstantProjectionRequirement(
	constant *types.Const,
	projection types.BasicKind,
) (DeclarationRequirement, error) {
	switch {
	case constant == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "constant projection constant is nil",
		}
	case !validConstantProjection(projection):
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "constant projection target representation is invalid",
		}
	}
	return DeclarationRequirement{
		owner:      MustSourceArtifactOwner(constant),
		kind:       DeclarationRequirementConstantProjection,
		projection: projection,
	}, nil
}

// NewLocalConstantProjectionRequirement requires one untyped constant declared
// inside a function to be projected once, at the given target basic
// representation, at its original lexical declaration. The enclosing function
// owns reconstruction because a function-local constant has no package
// declaration artifact; the dedup key is the
// (function, constant, representation) triple.
func NewLocalConstantProjectionRequirement(
	owner *types.Func,
	constant *types.Const,
	projection types.BasicKind,
) (DeclarationRequirement, error) {
	switch {
	case owner == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "local constant projection owner is nil",
		}
	case constant == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "local constant projection constant is nil",
		}
	case !validConstantProjection(projection):
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "local constant projection target representation is invalid",
		}
	}
	return DeclarationRequirement{
		owner:      MustSourceArtifactOwner(owner),
		kind:       DeclarationRequirementLocalConstantProjection,
		constant:   constant,
		projection: projection,
	}, nil
}

func validConstantProjection(projection types.BasicKind) bool {
	_, ok := ConstantProjectionType(projection)
	return ok
}

func NewDirectCallableControlRequirement(
	owner *types.Func,
	control CallableControlFacet,
) (DeclarationRequirement, error) {
	if owner == nil ||
		owner.Origin() != owner ||
		!control.Valid() ||
		control == CallableControlGoto ||
		control == CallableControlIteratorReturn {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "direct callable-control requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:   MustSourceArtifactOwner(owner),
		kind:    DeclarationRequirementCallableControl,
		control: control,
	}, nil
}

func NewCallableControlRequirement(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	control CallableControlFacet,
) (DeclarationRequirement, error) {
	if !validCallableControlAnchor(owner, enclosing, callable) ||
		!control.Valid() ||
		control == CallableControlGoto ||
		control == CallableControlIteratorReturn {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "callable-control requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     owner,
		kind:      DeclarationRequirementCallableControl,
		enclosing: enclosing,
		callable:  callable,
		control:   control,
	}, nil
}

func NewIteratorReturnControlRequirement(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	source *ast.RangeStmt,
) (DeclarationRequirement, error) {
	if !validCallableControlAnchor(owner, enclosing, callable) ||
		!validIteratorReturnRange(callable, source) {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "iterator-return control requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:        owner,
		kind:         DeclarationRequirementCallableControl,
		enclosing:    enclosing,
		callable:     callable,
		control:      CallableControlIteratorReturn,
		controlRange: source,
	}, nil
}

func NewGotoControlRequirement(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	label *types.Label,
	position token.Pos,
) (DeclarationRequirement, error) {
	if !validCallableControlAnchor(owner, enclosing, callable) ||
		label == nil ||
		!position.IsValid() ||
		position < callable.Pos() ||
		position > callable.End() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "goto control requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:           owner,
		kind:            DeclarationRequirementCallableControl,
		enclosing:       enclosing,
		callable:        callable,
		control:         CallableControlGoto,
		controlLabel:    label,
		controlPosition: position,
	}, nil
}

func validIteratorReturnRange(
	callable ast.Node,
	source *ast.RangeStmt,
) bool {
	return callable != nil &&
		source != nil &&
		source.X != nil &&
		source.Body != nil &&
		source.Pos() >= callable.Pos() &&
		source.End() <= callable.End()
}

func validCallableControlAnchor(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
) bool {
	if !owner.Valid() ||
		enclosing == nil ||
		callable == nil ||
		callable.Pos() < enclosing.Pos() ||
		callable.End() > enclosing.End() {
		return false
	}
	switch callable := callable.(type) {
	case *ast.FuncDecl:
		source, ok := owner.Source()
		function, functionOK := source.(*types.Func)
		return ok &&
			functionOK &&
			enclosing == callable &&
			callable.Type != nil &&
			callable.Body != nil &&
			function.Pos() >= callable.Pos() &&
			function.Pos() <= callable.End()
	case *ast.FuncLit:
		if callable.Type == nil || callable.Body == nil {
			return false
		}
		if source, ok := owner.Source(); ok {
			function, functionOK := source.(*types.Func)
			return functionOK &&
				function.Pos() >= enclosing.Pos() &&
				function.Pos() <= enclosing.End()
		}
		_, initializer, ok := owner.PackageInitializer()
		return ok &&
			initializer.Rhs != nil &&
			enclosing == initializer.Rhs
	default:
		return false
	}
}

func validCallableControlOwner(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
) bool {
	if enclosing != nil || callable != nil {
		return validCallableControlAnchor(owner, enclosing, callable)
	}
	source, ok := owner.Source()
	function, functionOK := source.(*types.Func)
	return ok && functionOK && function.Origin() == function
}

func NewEnvironmentBuiltinRequirement(
	builtin *types.Builtin,
	signature *types.Signature,
) (DeclarationRequirement, error) {
	if builtin == nil ||
		builtin.Pkg() == nil ||
		builtin.Parent() != builtin.Pkg().Scope() ||
		builtin.Parent().Lookup(builtin.Name()) != builtin ||
		!validEnvironmentBuiltinSignature(signature) {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "environment-builtin requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:                MustSourceArtifactOwner(builtin),
		kind:                 DeclarationRequirementEnvironmentBuiltin,
		environmentBuiltin:   builtin,
		environmentSignature: signature,
	}, nil
}

func NewEnvironmentBuiltinRequest(
	builtin *types.Builtin,
	signature *types.Signature,
) (RootRequest, error) {
	requirement, err := NewEnvironmentBuiltinRequirement(
		builtin,
		signature,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r DeclarationRequirement) EnvironmentBuiltin() (
	*types.Builtin,
	*types.Signature,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementEnvironmentBuiltin {
		return nil, nil, false
	}
	return r.environmentBuiltin, r.environmentSignature, true
}

func validEnvironmentBuiltinSignature(signature *types.Signature) bool {
	return signature != nil &&
		signature.Recv() == nil &&
		signature.RecvTypeParams().Len() == 0 &&
		signature.TypeParams().Len() == 0 &&
		signature.Params() != nil &&
		signature.Results() != nil
}
