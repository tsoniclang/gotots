package memorydescriptor

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	descriptor "github.com/tsoniclang/gotots/internal/emit/runtime/memorydescriptor"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Model struct {
	source types.Type
	word   int64
	slice  bool
}

func Resolve(context api.Context, source types.Type) (Model, bool) {
	if source == nil {
		return Model{}, false
	}
	slice := false
	switch underlying := source.Underlying().(type) {
	case *types.Slice:
		slice = true
	case *types.Basic:
		if underlying.Kind() != types.String {
			return Model{}, false
		}
	default:
		return Model{}, false
	}
	word := context.TypesSizes().Sizeof(types.Typ[types.UnsafePointer])
	fields := int64(2)
	if slice {
		fields++
	}
	if word != 4 && word != 8 || context.TypesSizes().Sizeof(types.Typ[types.Int]) != word ||
		context.TypesSizes().Alignof(types.Typ[types.Int]) != context.TypesSizes().Alignof(types.Typ[types.UnsafePointer]) ||
		context.TypesSizes().Sizeof(source) != fields*word ||
		context.TypesSizes().Alignof(source) != context.TypesSizes().Alignof(types.Typ[types.UnsafePointer]) {
		return Model{}, false
	}
	return Model{source: source, word: word, slice: slice}, true
}

func (model Model) SourceType() types.Type {
	return model.source
}

func (model Model) WordBytes() int64 {
	return model.word
}

func (model Model) IsSlice() bool {
	return model.slice
}

func (model Model) StorageType(context api.Context) (api.TypeEmission, error) {
	symbol := api.RuntimeStringHeader32
	if model.slice {
		symbol = api.RuntimeSliceHeader32
	}
	if model.word == 8 {
		symbol = api.RuntimeStringHeader64
		if model.slice {
			symbol = api.RuntimeSliceHeader64
		}
	}
	if model.source == nil || model.word != 4 && model.word != 8 {
		return api.TypeEmission{}, &api.InvariantError{Role: context.Role(), Reason: "memory descriptor has no selected source ABI"}
	}
	reference, err := context.Names().Runtime(symbol, api.ImportPhaseType)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(context.Factory().TypeQueryNode(reference.EntityName(context.Factory()), nil), reference.Requests()...), nil
}

func (model Model) WordType(context api.Context) (api.TypeEmission, error) {
	symbol := tsoniccore.SymbolInt32
	if model.word == 8 {
		symbol = tsoniccore.SymbolInt64
	} else if model.word != 4 {
		return api.TypeEmission{}, &api.InvariantError{Role: context.Role(), Reason: "memory descriptor word width is invalid"}
	}
	reference, err := context.Names().TsonicCore(symbol)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(context.Factory().TypeReferenceNode(reference.EntityName(context.Factory()), nil), reference.Requests()...), nil
}

func (model Model) ZeroWord(factory tsgo.Factory) tsgo.Expression {
	if model.word == 8 {
		return factory.BigIntLiteral("0n", tsgo.TokenFlagsNone)
	}
	return factory.NumericLiteral("0", tsgo.TokenFlagsNone)
}

func (model Model) Zero(context api.Context) (api.ExpressionEmission, error) {
	storage, err := model.StorageType(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	factory := context.Factory()
	fieldType := func(name string) tsgo.TypeNode {
		return factory.IndexedAccessTypeNode(storage.Value(), factory.LiteralTypeNode(factory.StringLiteral(name, tsgo.TokenFlagsNone)))
	}
	fields := []tsgo.ObjectLiteralElementLike{
		factory.PropertyAssignment(nil, factory.Identifier(descriptor.DataMember), nil, fieldType(descriptor.DataMember), factory.VoidExpression(factory.NumericLiteral("0", tsgo.TokenFlagsNone))),
		factory.PropertyAssignment(nil, factory.Identifier(descriptor.LengthMember), nil, fieldType(descriptor.LengthMember), model.ZeroWord(factory)),
	}
	if model.slice {
		fields = append(fields, factory.PropertyAssignment(nil, factory.Identifier(descriptor.CapacityMember), nil, fieldType(descriptor.CapacityMember), model.ZeroWord(factory)))
	}
	return api.DirectExpression(factory.AsExpression(factory.ObjectLiteralExpression(fields, false), storage.Value()), storage.Requests()...), nil
}
