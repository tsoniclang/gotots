package artifact

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ProjectSourceContract(
	factory tsgo.Factory,
	publicName string,
	additionalBindings []string,
	statements []tsgo.Statement,
) (Contract, error) {
	contract, err := ProjectContract(factory, statements)
	if err != nil {
		return Contract{}, err
	}
	exportSet := make(map[string]struct{}, len(additionalBindings)+1)
	if sourceBindingPresent(publicName, statements) {
		exportSet[publicName] = struct{}{}
	}
	for _, binding := range additionalBindings {
		if binding == "" || !sourceBindingPresent(binding, statements) {
			return Contract{}, &ContractError{
				Reason: "additional package binding has no target declaration",
			}
		}
		if _, duplicate := exportSet[binding]; duplicate {
			return Contract{}, &ContractError{
				Reason: "package binding is duplicated",
			}
		}
		exportSet[binding] = struct{}{}
	}
	exports := make([]string, 0, len(exportSet))
	for binding := range exportSet {
		exports = append(exports, binding)
	}
	sort.Strings(exports)
	encoded, err := tsgo.EncodeNode(factory.NamedExports(
		exportSpecifiers(factory, exports),
	))
	if err != nil {
		return Contract{}, err
	}
	contract, err = contract.withOwnedFacet(
		api.ArtifactFacetExportSurface,
		encoded,
	)
	if err != nil {
		return Contract{}, err
	}
	return contract.withOwnedExports(exports)
}

func sourceBindingPresent(
	name string,
	statements []tsgo.Statement,
) bool {
	if name == "" {
		return false
	}
	for _, statement := range statements {
		switch statement := statement.(type) {
		case tsgo.FunctionDeclaration:
			if statement.Name() != nil && statement.Name().Text() == name {
				return true
			}
		case tsgo.ClassDeclaration:
			if statement.Name() != nil && statement.Name().Text() == name {
				return true
			}
		case tsgo.EnumDeclaration:
			if statement.Name() != nil && statement.Name().Text() == name {
				return true
			}
		case tsgo.InterfaceDeclaration:
			if statement.Name() != nil && statement.Name().Text() == name {
				return true
			}
		case tsgo.TypeAliasDeclaration:
			if statement.Name() != nil && statement.Name().Text() == name {
				return true
			}
		case tsgo.VariableStatement:
			for _, declaration := range statement.DeclarationList().Declarations() {
				identifier, ok := declaration.Name().(tsgo.Identifier)
				if ok && identifier.Text() == name {
					return true
				}
			}
		}
	}
	return false
}

func exportSpecifiers(
	factory tsgo.Factory,
	names []string,
) []tsgo.ExportSpecifier {
	result := make([]tsgo.ExportSpecifier, len(names))
	for index, name := range names {
		result[index] = factory.ExportSpecifier(
			false,
			nil,
			factory.Identifier(name),
		)
	}
	return result
}
