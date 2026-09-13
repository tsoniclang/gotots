package memorydescriptor

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	DataMember     = "data"
	LengthMember   = "length"
	CapacityMember = "capacity"
)

func Markers(symbol api.RuntimeSymbol) ([]tsoniccore.Symbol, error) {
	word, _, err := selection(symbol)
	if err != nil {
		return nil, err
	}
	return []tsoniccore.Symbol{tsoniccore.SymbolStruct, tsoniccore.SymbolField, tsoniccore.SymbolRawPointer, word}, nil
}

func Build(factory tsgo.Factory, symbol api.RuntimeSymbol) (tsgo.Statement, error) {
	wordSymbol, slice, err := selection(symbol)
	if err != nil {
		return nil, err
	}
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return nil, err
	}
	word, err := tsoniccore.Resolve(wordSymbol)
	if err != nil {
		return nil, err
	}
	pointer, err := tsoniccore.Resolve(tsoniccore.SymbolRawPointer)
	if err != nil {
		return nil, err
	}
	structure, err := tsoniccore.Resolve(tsoniccore.SymbolStruct)
	if err != nil {
		return nil, err
	}
	field, err := tsoniccore.Resolve(tsoniccore.SymbolField)
	if err != nil {
		return nil, err
	}
	wordType := factory.TypeReferenceNode(factory.Identifier(word.Export()), nil)
	dataType := factory.UnionTypeNode([]tsgo.TypeNode{
		factory.TypeReferenceNode(factory.Identifier(pointer.Export()), nil),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
	})
	type member struct {
		name string
		kind tsgo.TypeNode
	}
	members := []member{{DataMember, dataType}, {LengthMember, wordType}}
	if slice {
		members = append(members, member{CapacityMember, wordType})
	}
	fields := make([]tsgo.TypeElement, 0, len(members))
	values := make([]tsgo.ObjectLiteralElementLike, 0, len(members))
	for _, member := range members {
		fields = append(fields, factory.PropertySignatureDeclaration(nil, factory.Identifier(member.name), nil, member.kind, factory.OmittedExpression()))
		values = append(values, factory.PropertyAssignment(nil, factory.Identifier(member.name), nil, member.kind,
			factory.CallExpression(factory.Identifier(field.Export()), nil, []tsgo.TypeNode{member.kind}, nil, tsgo.NodeFlagsNone)))
	}
	return factory.VariableStatement([]tsgo.ModifierLike{factory.ExportKeyword()}, factory.VariableDeclarationList(
		[]tsgo.VariableDeclaration{factory.VariableDeclaration(factory.Identifier(contract.ExportedName()), nil,
			factory.TypeLiteralNode(fields), factory.CallExpression(factory.Identifier(structure.Export()), nil, nil,
				[]tsgo.Expression{factory.ObjectLiteralExpression(values, true)}, tsgo.NodeFlagsNone))}, tsgo.NodeFlagsConst)), nil
}

func selection(symbol api.RuntimeSymbol) (tsoniccore.Symbol, bool, error) {
	switch symbol {
	case api.RuntimeSliceHeader32:
		return tsoniccore.SymbolInt32, true, nil
	case api.RuntimeSliceHeader64:
		return tsoniccore.SymbolInt64, true, nil
	case api.RuntimeStringHeader32:
		return tsoniccore.SymbolInt32, false, nil
	case api.RuntimeStringHeader64:
		return tsoniccore.SymbolInt64, false, nil
	default:
		return tsoniccore.SymbolInvalid, false, &api.RuntimeSymbolError{Symbol: symbol}
	}
}
