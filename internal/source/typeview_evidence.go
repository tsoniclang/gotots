package source

import (
	"fmt"
	"go/ast"
	"go/types"
)

// VisitDefinitions visits the direct checker definition relation without
// exposing or copying its mutable map. Consumers key their results by
// canonical occurrence identity, so map iteration order has no authority.
func (v *TypeInfoView) VisitDefinitions(
	visit func(*ast.Ident, types.Object) error,
) error {
	if v == nil || v.info == nil {
		return fmt.Errorf("checker definition evidence is absent")
	}
	if visit == nil {
		return fmt.Errorf("checker definition visitor is absent")
	}
	for identifier, object := range v.info.Defs {
		if identifier == nil || object == nil {
			continue
		}
		if err := visit(identifier, object); err != nil {
			return err
		}
	}
	return nil
}

// VisitImplicits visits the direct checker implicit-definition relation without
// exposing or copying its mutable map.
func (v *TypeInfoView) VisitImplicits(
	visit func(ast.Node, types.Object) error,
) error {
	if v == nil || v.info == nil {
		return fmt.Errorf("checker implicit evidence is absent")
	}
	if visit == nil {
		return fmt.Errorf("checker implicit visitor is absent")
	}
	for node, object := range v.info.Implicits {
		if node == nil || object == nil {
			continue
		}
		if err := visit(node, object); err != nil {
			return err
		}
	}
	return nil
}

// VisitScopes visits the direct checker scope relation without exposing or
// copying its mutable map.
func (v *TypeInfoView) VisitScopes(
	visit func(ast.Node, *types.Scope) error,
) error {
	if v == nil || v.info == nil {
		return fmt.Errorf("checker scope evidence is absent")
	}
	if visit == nil {
		return fmt.Errorf("checker scope visitor is absent")
	}
	for node, scope := range v.info.Scopes {
		if node == nil || scope == nil {
			continue
		}
		if err := visit(node, scope); err != nil {
			return err
		}
	}
	return nil
}
