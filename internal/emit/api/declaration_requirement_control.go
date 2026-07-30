package api

import (
	"go/ast"
	"go/token"
	"go/types"
)

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
