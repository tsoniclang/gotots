package stagecheck

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

func (
	verifier *checkerSemanticVerifier,
) independentCompileTimeContext(
	occurrence structure.Occurrence,
) bool {
	return !verifier.independentCompileTimeAnchor(
		occurrence,
	).IsZero()
}

func (
	verifier *checkerSemanticVerifier,
) independentCompileTimeAnchor(
	occurrence structure.Occurrence,
) identity.OccurrenceID {
	id := occurrence.ID()
	if verifier.compileTimeResolved[id] {
		return verifier.compileTimeAnchor[id]
	}
	var anchor identity.OccurrenceID
	if occurrence.Role() == catalog.RoleArrayLength ||
		(occurrence.Role() == catalog.RoleInitializerValue &&
			verifier.independentConstInitializer(occurrence)) {
		anchor = occurrence.ID()
	}
	if anchor.IsZero() && !occurrence.Parent().IsZero() {
		if parent, present :=
			verifier.expected.occurrences.get(occurrence.Parent()); present {
			anchor =
				verifier.independentCompileTimeAnchor(parent.Occurrence)
		}
	}
	verifier.compileTimeAnchor[id] = anchor
	verifier.compileTimeResolved[id] = true
	return anchor
}

func (
	verifier *checkerSemanticVerifier,
) independentCompileTimeCoverage(
	occurrence structure.Occurrence,
) (types.Object, types.Type) {
	anchorID := verifier.independentCompileTimeAnchor(occurrence)
	if anchorID.IsZero() {
		return nil, nil
	}
	anchor := verifier.expected.occurrence(anchorID)
	parent, present := verifier.expected.occurrences.get(anchor.Parent())
	if !present {
		return nil, nil
	}
	node, present := verifier.index.OccurrenceNode(parent.ID())
	if !present {
		return nil, nil
	}
	switch anchor.Role() {
	case catalog.RoleArrayLength:
		array, ok := node.(*ast.ArrayType)
		if !ok {
			return nil, nil
		}
		value, present := verifier.view.TypeOf(array)
		if !present {
			return nil, nil
		}
		return nil, value.Type
	case catalog.RoleInitializerValue:
		valueSpec, ok := node.(*ast.ValueSpec)
		if !ok || anchor.Ordinal() >= len(valueSpec.Names) {
			return nil, nil
		}
		object, _ := verifier.view.DefOf(
			valueSpec.Names[anchor.Ordinal()],
		)
		return object, nil
	default:
		return nil, nil
	}
}

func (
	verifier *checkerSemanticVerifier,
) independentConstInitializer(
	initializer structure.Occurrence,
) bool {
	valueSpecOccurrence, present :=
		verifier.expected.occurrences.get(initializer.Parent())
	if !present {
		return false
	}
	valueSpecNode, present := verifier.index.OccurrenceNode(
		valueSpecOccurrence.ID(),
	)
	if !present {
		return false
	}
	if _, ok := valueSpecNode.(*ast.ValueSpec); !ok {
		return false
	}
	genDeclOccurrence, present :=
		verifier.expected.occurrences.get(valueSpecOccurrence.Parent())
	if !present {
		return false
	}
	genDeclNode, present := verifier.index.OccurrenceNode(
		genDeclOccurrence.ID(),
	)
	if !present {
		return false
	}
	genDecl, ok := genDeclNode.(*ast.GenDecl)
	return ok && genDecl.Tok == token.CONST
}
