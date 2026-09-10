package complex

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func StorageMarkers(symbol api.RuntimeSymbol) []tsoniccore.Symbol {
	component := tsoniccore.SymbolInvalid
	switch symbol {
	case api.RuntimeComplex64Storage:
		component = tsoniccore.SymbolFloat32
	case api.RuntimeComplex128Storage:
		component = tsoniccore.SymbolFloat64
	default:
		return nil
	}
	return []tsoniccore.Symbol{tsoniccore.SymbolStruct, tsoniccore.SymbolField, component}
}

func storageDefinition(factory tsgo.Factory, symbol api.RuntimeSymbol, name string) (tsgo.Statement, bool) {
	markers := StorageMarkers(symbol)
	if len(markers) != 0 {
		structure, err := tsoniccore.Resolve(markers[0])
		if err != nil {
			return nil, false
		}
		field, err := tsoniccore.Resolve(markers[1])
		if err != nil {
			return nil, false
		}
		component, err := tsoniccore.Resolve(markers[2])
		if err != nil {
			return nil, false
		}
		componentType := factory.TypeReferenceNode(factory.Identifier(component.Export()), nil)
		var fields []tsgo.TypeElement
		var values []tsgo.ObjectLiteralElementLike
		for _, member := range []string{RealMember, ImagMember} {
			fields = append(fields, factory.PropertySignatureDeclaration(nil, factory.Identifier(member), nil, componentType, factory.OmittedExpression()))
			values = append(values, factory.PropertyAssignment(nil, factory.Identifier(member), nil, componentType,
				factory.CallExpression(factory.Identifier(field.Export()), nil, []tsgo.TypeNode{componentType}, nil, tsgo.NodeFlagsNone)))
		}
		return factory.VariableStatement([]tsgo.ModifierLike{factory.ExportKeyword()}, factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(factory.Identifier(name), nil, factory.TypeLiteralNode(fields),
				factory.CallExpression(factory.Identifier(structure.Export()), nil, nil, []tsgo.Expression{factory.ObjectLiteralExpression(values, true)}, tsgo.NodeFlagsNone))},
			tsgo.NodeFlagsConst)), true
	}
	var classSymbol, storageSymbol api.RuntimeSymbol
	toStorage := false
	switch symbol {
	case api.RuntimeComplex64ToStorage, api.RuntimeComplex64FromStorage:
		classSymbol, storageSymbol = api.RuntimeComplex64, api.RuntimeComplex64Storage
		toStorage = symbol == api.RuntimeComplex64ToStorage
	case api.RuntimeComplex128ToStorage, api.RuntimeComplex128FromStorage:
		classSymbol, storageSymbol = api.RuntimeComplex128, api.RuntimeComplex128Storage
		toStorage = symbol == api.RuntimeComplex128ToStorage
	default:
		return nil, false
	}
	class, err := api.RuntimeContract(classSymbol)
	if err != nil {
		return nil, false
	}
	storage, err := api.RuntimeContract(storageSymbol)
	if err != nil {
		return nil, false
	}
	target := builder{factory: factory}
	logicalType := target.typeReference(class.ExportedName())
	physicalType := factory.TypeQueryNode(target.id(storage.ExportedName()), nil)
	input, output := tsgo.TypeNode(physicalType), logicalType
	components := []tsgo.Expression{target.property(target.id("value"), RealMember), target.property(target.id("value"), ImagMember)}
	result := tsgo.Expression(target.call(target.property(target.id(class.ExportedName()), MakeMember), components...))
	if toStorage {
		input, output = logicalType, physicalType
		result = factory.ObjectLiteralExpression([]tsgo.ObjectLiteralElementLike{
			factory.PropertyAssignment(nil, target.id(RealMember), nil, target.numberType(), components[0]),
			factory.PropertyAssignment(nil, target.id(ImagMember), nil, target.numberType(), components[1]),
		}, false)
	}
	return factory.FunctionDeclaration([]tsgo.ModifierLike{factory.ExportKeyword()}, nil, target.id(name), nil,
		[]tsgo.ParameterDeclaration{target.parameter("value", input)}, output,
		factory.Block([]tsgo.Statement{factory.ReturnStatement(result)}, true)), true
}
