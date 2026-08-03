package cooperative

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
)

type genericTypePair struct {
	declaration  types.Type
	instantiated types.Type
}

type callableCorrespondence struct {
	seen                map[genericTypePair]struct{}
	leaf                func(*types.Signature, *types.Signature) error
	traverseIdentical   bool
	stopAtNamedBoundary bool
}

func (c *callableCorrespondence) signatureMembers(
	declaration *types.Signature,
	instantiated *types.Signature,
) error {
	if declaration == nil || instantiated == nil ||
		declaration.Params().Len() != instantiated.Params().Len() ||
		declaration.Results().Len() != instantiated.Results().Len() ||
		declaration.Variadic() != instantiated.Variadic() {
		return c.invalid()
	}
	for index := range declaration.Params().Len() {
		if err := c.pair(
			declaration.Params().At(index).Type(),
			instantiated.Params().At(index).Type(),
		); err != nil {
			return err
		}
	}
	for index := range declaration.Results().Len() {
		if err := c.pair(
			declaration.Results().At(index).Type(),
			instantiated.Results().At(index).Type(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (c *callableCorrespondence) pair(
	declaration types.Type,
	instantiated types.Type,
) error {
	if declaration == nil || instantiated == nil {
		return c.invalid()
	}
	declaration = types.Unalias(declaration)
	instantiated = types.Unalias(instantiated)
	if _, parameter := declaration.(*types.TypeParam); parameter {
		return nil
	}
	pair := genericTypePair{declaration: declaration, instantiated: instantiated}
	if _, visited := c.seen[pair]; visited {
		return nil
	}
	c.seen[pair] = struct{}{}

	declarationCallable, declarationIsCallable := callable.Signature(declaration)
	if declarationIsCallable {
		instantiatedCallable, instantiatedIsCallable := callable.Signature(instantiated)
		if !instantiatedIsCallable {
			return c.invalid()
		}
		if c.leaf != nil {
			if err := c.leaf(declarationCallable, instantiatedCallable); err != nil {
				return err
			}
		}
		return c.signatureMembers(declarationCallable, instantiatedCallable)
	}
	identical := types.Identical(declaration, instantiated)
	if identical && !c.traverseIdentical {
		return nil
	}

	switch declaration := declaration.(type) {
	case *types.Array:
		instantiated, ok := instantiated.(*types.Array)
		if !ok || declaration.Len() != instantiated.Len() {
			return c.invalid()
		}
		return c.pair(declaration.Elem(), instantiated.Elem())
	case *types.Slice:
		instantiated, ok := instantiated.(*types.Slice)
		if !ok {
			return c.invalid()
		}
		return c.pair(declaration.Elem(), instantiated.Elem())
	case *types.Pointer:
		instantiated, ok := instantiated.(*types.Pointer)
		if !ok {
			return c.invalid()
		}
		return c.pair(declaration.Elem(), instantiated.Elem())
	case *types.Map:
		instantiated, ok := instantiated.(*types.Map)
		if !ok {
			return c.invalid()
		}
		if err := c.pair(declaration.Key(), instantiated.Key()); err != nil {
			return err
		}
		return c.pair(declaration.Elem(), instantiated.Elem())
	case *types.Chan:
		instantiated, ok := instantiated.(*types.Chan)
		if !ok || declaration.Dir() != instantiated.Dir() {
			return c.invalid()
		}
		return c.pair(declaration.Elem(), instantiated.Elem())
	case *types.Named:
		instantiated, ok := instantiated.(*types.Named)
		if !ok || declaration.Origin() != instantiated.Origin() {
			return c.invalid()
		}
		if !c.stopAtNamedBoundary {
			return c.pair(declaration.Underlying(), instantiated.Underlying())
		}
		if declaration.TypeArgs().Len() != instantiated.TypeArgs().Len() {
			return c.invalid()
		}
		for index := range declaration.TypeArgs().Len() {
			if err := c.pair(
				declaration.TypeArgs().At(index),
				instantiated.TypeArgs().At(index),
			); err != nil {
				return err
			}
		}
		return nil
	case *types.Struct:
		instantiated, ok := instantiated.(*types.Struct)
		if !ok || declaration.NumFields() != instantiated.NumFields() {
			return c.invalid()
		}
		for index := range declaration.NumFields() {
			if declaration.Field(index).Id() != instantiated.Field(index).Id() ||
				declaration.Field(index).Embedded() != instantiated.Field(index).Embedded() ||
				declaration.Tag(index) != instantiated.Tag(index) {
				return c.invalid()
			}
			if err := c.pair(
				declaration.Field(index).Type(),
				instantiated.Field(index).Type(),
			); err != nil {
				return err
			}
		}
		return nil
	case *types.Tuple:
		instantiated, ok := instantiated.(*types.Tuple)
		if !ok || declaration.Len() != instantiated.Len() {
			return c.invalid()
		}
		for index := range declaration.Len() {
			if err := c.pair(
				declaration.At(index).Type(),
				instantiated.At(index).Type(),
			); err != nil {
				return err
			}
		}
		return nil
	case *types.Interface:
		instantiated, ok := instantiated.(*types.Interface)
		if !ok {
			return c.invalid()
		}
		declaration = declaration.Complete()
		instantiated = instantiated.Complete()
		if declaration.NumMethods() != instantiated.NumMethods() {
			return c.invalid()
		}
		for index := range declaration.NumMethods() {
			declarationMethod := declaration.Method(index)
			instantiatedMethod := instantiated.Method(index)
			if declarationMethod.Id() != instantiatedMethod.Id() {
				return c.invalid()
			}
			declarationSignature, declarationOK := declarationMethod.Type().(*types.Signature)
			instantiatedSignature, instantiatedOK := instantiatedMethod.Type().(*types.Signature)
			if !declarationOK || !instantiatedOK {
				return c.invalid()
			}
			if err := c.signatureMembers(declarationSignature, instantiatedSignature); err != nil {
				return err
			}
		}
		return nil
	case *types.Basic:
		if !identical {
			return c.invalid()
		}
	}
	return nil
}

func (c *callableCorrespondence) invalid() error {
	return &api.InvariantError{
		Role:   api.RoleCallArgument,
		Reason: "generic callable correspondence changed structural shape",
	}
}
