package artifact

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ProjectExplicitValueContract(
	factory tsgo.Factory,
	statement tsgo.Statement,
	targetType tsgo.TypeNode,
) (Contract, error) {
	variable, ok := statement.(tsgo.VariableStatement)
	if !ok || targetType == nil {
		return Contract{}, &ContractError{
			Reason: "explicit value contract is not a typed variable",
		}
	}
	declarations := variable.DeclarationList().Declarations()
	if len(declarations) != 1 || declarations[0].Type() != nil ||
		declarations[0].Initializer() == nil {
		return Contract{}, &ContractError{
			Kind:   variable.Kind(),
			Reason: "explicit value contract has no one inferred implementation",
		}
	}
	name, ok := declarations[0].Name().(tsgo.Identifier)
	if !ok || name.Text() == "" {
		return Contract{}, &ContractError{
			Kind:   declarations[0].Kind(),
			Reason: "explicit value contract name is not an identifier",
		}
	}
	projected := factory.VariableStatement(
		variable.Modifiers(),
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(
				name,
				declarations[0].ExclamationToken(),
				targetType,
				nil,
			)},
			variable.DeclarationList().Flags(),
		),
	)
	encodedValue, err := tsgo.EncodeNode(projected)
	if err != nil {
		return Contract{}, err
	}
	exports := factory.NamedExports([]tsgo.ExportSpecifier{
		factory.ExportSpecifier(false, nil, name),
	})
	encodedExports, err := tsgo.EncodeNode(exports)
	if err != nil {
		return Contract{}, err
	}
	contract, err := NewContract().withOwnedFacet(
		api.ArtifactFacetValueSurface,
		encodedValue,
	)
	if err != nil {
		return Contract{}, err
	}
	contract, err = contract.withOwnedFacet(
		api.ArtifactFacetExportSurface,
		encodedExports,
	)
	if err != nil {
		return Contract{}, err
	}
	return contract.withOwnedExports([]string{name.Text()})
}
