package artifact

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type ContractError struct {
	Kind   tsgo.SyntaxKind
	Reason string
}

func (e *ContractError) Error() string {
	return fmt.Sprintf(
		"project observable contract for TS-Go kind %d: %s",
		e.Kind,
		e.Reason,
	)
}

func ProjectContract(
	factory tsgo.Factory,
	statements []tsgo.Statement,
) (Contract, error) {
	if len(statements) == 0 {
		return Contract{}, &ContractError{
			Reason: "materialized artifact has no target declaration",
		}
	}
	nodes := make(map[api.ArtifactFacet][]tsgo.Node)
	var exports []string
	for _, statement := range statements {
		switch statement := statement.(type) {
		case tsgo.FunctionDeclaration:
			if statement.Type() == nil {
				return Contract{}, &ContractError{
					Kind:   statement.Kind(),
					Reason: "callable surface relies on target inference",
				}
			}
			nodes[api.ArtifactFacetCallableSignature] = append(
				nodes[api.ArtifactFacetCallableSignature],
				factory.FunctionDeclaration(
					statement.Modifiers(),
					statement.AsteriskToken(),
					statement.Name(),
					statement.TypeParameters(),
					statement.Parameters(),
					statement.Type(),
					nil,
				),
			)
			exports = append(exports, statement.Name().Text())
		case tsgo.ClassDeclaration:
			classFacets, err := projectClassContract(factory, statement)
			if err != nil {
				return Contract{}, err
			}
			for facet, node := range classFacets {
				nodes[facet] = append(nodes[facet], node)
			}
			exports = append(exports, statement.Name().Text())
		case tsgo.InterfaceDeclaration:
			nodes[api.ArtifactFacetInstanceTypeSurface] = append(
				nodes[api.ArtifactFacetInstanceTypeSurface],
				statement,
			)
			exports = append(exports, statement.Name().Text())
		case tsgo.TypeAliasDeclaration:
			nodes[api.ArtifactFacetInstanceTypeSurface] = append(
				nodes[api.ArtifactFacetInstanceTypeSurface],
				statement,
			)
			exports = append(exports, statement.Name().Text())
		case tsgo.VariableStatement:
			declarations := statement.DeclarationList().Declarations()
			projected := make([]tsgo.VariableDeclaration, len(declarations))
			for index, declaration := range declarations {
				if declaration.Type() == nil {
					return Contract{}, &ContractError{
						Kind:   declaration.Kind(),
						Reason: "value surface relies on target inference",
					}
				}
				name, ok := declaration.Name().(tsgo.Identifier)
				if !ok {
					return Contract{}, &ContractError{
						Kind:   declaration.Kind(),
						Reason: "observable value binding name is not an identifier",
					}
				}
				exports = append(exports, name.Text())
				projected[index] = factory.VariableDeclaration(
					declaration.Name(),
					declaration.ExclamationToken(),
					declaration.Type(),
					nil,
				)
			}
			nodes[api.ArtifactFacetValueSurface] = append(
				nodes[api.ArtifactFacetValueSurface],
				factory.VariableStatement(
					statement.Modifiers(),
					factory.VariableDeclarationList(
						projected,
						statement.DeclarationList().Flags(),
					),
				),
			)
		default:
			kind := tsgo.SyntaxKind(0)
			if statement != nil {
				kind = statement.Kind()
			}
			return Contract{}, &ContractError{
				Kind:   kind,
				Reason: "declaration form has no closed observable projection",
			}
		}
	}
	sort.Strings(exports)
	exportSpecifiers := make([]tsgo.ExportSpecifier, len(exports))
	for index, name := range exports {
		exportSpecifiers[index] = factory.ExportSpecifier(
			false,
			nil,
			factory.Identifier(name),
		)
	}
	nodes[api.ArtifactFacetExportSurface] = []tsgo.Node{
		factory.NamedExports(exportSpecifiers),
	}
	contract := NewContract()
	for facet, facetNodes := range nodes {
		encoded, err := tsgo.EncodeNode(factory.SyntaxList(facetNodes))
		if err != nil {
			return Contract{}, err
		}
		contract, err = contract.withOwnedFacet(facet, encoded)
		if err != nil {
			return Contract{}, err
		}
	}
	contract, err := contract.withOwnedExports(exports)
	if err != nil {
		return Contract{}, err
	}
	if contract.present == 0 {
		return Contract{}, &ContractError{
			Reason: "artifact contains no observable declaration",
		}
	}
	return contract, nil
}

func ProjectCoverageContract(
	factory tsgo.Factory,
	statements []tsgo.Statement,
) (Contract, error) {
	if len(statements) != 0 {
		return Contract{}, &ContractError{
			Reason: "coverage-only artifact contains target declarations",
		}
	}
	encoded, err := tsgo.EncodeNode(factory.NamedExports(nil))
	if err != nil {
		return Contract{}, err
	}
	contract, err := NewContract().withOwnedFacet(
		api.ArtifactFacetExportSurface,
		encoded,
	)
	if err != nil {
		return Contract{}, err
	}
	return contract.withOwnedExports(nil)
}

func ProjectFacet(
	facet api.ArtifactFacet,
	node tsgo.Node,
) (Contract, error) {
	if facet == api.ArtifactFacetExportSurface {
		kind := tsgo.SyntaxKind(0)
		if node != nil {
			kind = node.Kind()
		}
		return Contract{}, &ContractError{
			Kind:   kind,
			Reason: "export surface must be projected from complete declarations",
		}
	}
	encoded, err := tsgo.EncodeNode(node)
	if err != nil {
		return Contract{}, err
	}
	return NewContract().withOwnedFacet(facet, encoded)
}

func ProjectClassMemberContract(
	factory tsgo.Factory,
	members []tsgo.ClassElement,
) (Contract, error) {
	if len(members) == 0 {
		return Contract{}, &ContractError{
			Reason: "class-member artifact has no target member",
		}
	}
	signatures := make([]tsgo.Node, 0, len(members))
	implementations := make([]tsgo.Node, 0, len(members))
	for _, member := range members {
		projected, observable, err := projectClassMember(factory, member)
		if err != nil {
			return Contract{}, err
		}
		if !observable {
			kind := tsgo.SyntaxKind(0)
			if member != nil {
				kind = member.Kind()
			}
			return Contract{}, &ContractError{
				Kind:   kind,
				Reason: "class-member contribution has no observable signature",
			}
		}
		signatures = append(signatures, projected)
		implementations = append(implementations, member)
	}
	signature, err := tsgo.EncodeNode(factory.SyntaxList(signatures))
	if err != nil {
		return Contract{}, err
	}
	implementation, err := tsgo.EncodeNode(
		factory.SyntaxList(implementations),
	)
	if err != nil {
		return Contract{}, err
	}
	contract, err := NewContract().withOwnedFacet(
		api.ArtifactFacetCallableSignature,
		signature,
	)
	if err != nil {
		return Contract{}, err
	}
	return contract.withOwnedFacet(
		api.ArtifactFacetImplementation,
		implementation,
	)
}

func projectClassContract(
	factory tsgo.Factory,
	class tsgo.ClassDeclaration,
) (map[api.ArtifactFacet]tsgo.Node, error) {
	constructors := make([]tsgo.ClassElement, 0)
	instance := make([]tsgo.ClassElement, 0)
	static := make([]tsgo.ClassElement, 0)
	for _, member := range class.Members() {
		projected, observable, err := projectClassMember(factory, member)
		if err != nil {
			return nil, err
		}
		if !observable {
			continue
		}
		if _, ok := projected.(tsgo.ConstructorDeclaration); ok {
			constructors = append(constructors, projected)
			continue
		}
		if classMemberIsStatic(projected) {
			static = append(static, projected)
		} else {
			instance = append(instance, projected)
		}
	}
	return map[api.ArtifactFacet]tsgo.Node{
		api.ArtifactFacetConstructorSurface: factory.ClassDeclaration(
			class.Modifiers(),
			class.Name(),
			class.TypeParameters(),
			class.HeritageClauses(),
			constructors,
		),
		api.ArtifactFacetInstanceTypeSurface: factory.ClassDeclaration(
			class.Modifiers(),
			class.Name(),
			class.TypeParameters(),
			class.HeritageClauses(),
			instance,
		),
		api.ArtifactFacetStaticSurface: factory.ClassDeclaration(
			class.Modifiers(),
			class.Name(),
			class.TypeParameters(),
			class.HeritageClauses(),
			static,
		),
	}, nil
}

func projectClassMember(
	factory tsgo.Factory,
	member tsgo.ClassElement,
) (tsgo.ClassElement, bool, error) {
	switch member := member.(type) {
	case tsgo.PropertyDeclaration:
		if member.Type() == nil {
			return nil, false, &ContractError{
				Kind:   member.Kind(),
				Reason: "property surface relies on target inference",
			}
		}
		return factory.PropertyDeclaration(
			member.Modifiers(),
			member.Name(),
			member.PostfixToken(),
			member.Type(),
			nil,
		), true, nil
	case tsgo.MethodDeclaration:
		if member.Type() == nil {
			return nil, false, &ContractError{
				Kind:   member.Kind(),
				Reason: "method surface relies on target inference",
			}
		}
		return factory.MethodDeclaration(
			member.Modifiers(),
			member.AsteriskToken(),
			member.Name(),
			member.PostfixToken(),
			member.TypeParameters(),
			member.Parameters(),
			member.Type(),
			nil,
		), true, nil
	case tsgo.ConstructorDeclaration:
		return factory.ConstructorDeclaration(
			member.Modifiers(),
			member.TypeParameters(),
			member.Parameters(),
			member.Type(),
			nil,
		), true, nil
	case tsgo.GetAccessorDeclaration:
		if member.Type() == nil {
			return nil, false, &ContractError{
				Kind:   member.Kind(),
				Reason: "getter surface relies on target inference",
			}
		}
		return factory.GetAccessorDeclaration(
			member.Modifiers(),
			member.Name(),
			member.TypeParameters(),
			member.Parameters(),
			member.Type(),
			nil,
		), true, nil
	case tsgo.SetAccessorDeclaration:
		return factory.SetAccessorDeclaration(
			member.Modifiers(),
			member.Name(),
			member.TypeParameters(),
			member.Parameters(),
			member.Type(),
			nil,
		), true, nil
	case tsgo.ClassStaticBlockDeclaration:
		return nil, false, nil
	default:
		kind := tsgo.SyntaxKind(0)
		if member != nil {
			kind = member.Kind()
		}
		return nil, false, &ContractError{
			Kind:   kind,
			Reason: "class member has no closed observable projection",
		}
	}
}

func classMemberIsStatic(member tsgo.ClassElement) bool {
	var modifiers []tsgo.ModifierLike
	switch member := member.(type) {
	case tsgo.PropertyDeclaration:
		modifiers = member.Modifiers()
	case tsgo.MethodDeclaration:
		modifiers = member.Modifiers()
	case tsgo.GetAccessorDeclaration:
		modifiers = member.Modifiers()
	case tsgo.SetAccessorDeclaration:
		modifiers = member.Modifiers()
	default:
		return false
	}
	for _, modifier := range modifiers {
		if modifier.Kind() == tsgo.SyntaxKindStaticKeyword {
			return true
		}
	}
	return false
}
