package memory

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	memorymarker "github.com/tsoniclang/gotots/internal/emit/marker/memory"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/emit/value/structconstruction"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) memoryRecord(context api.Context, source ast.Node, sourceType types.Type, structure *types.Struct, value api.ExpressionEmission, toMemory bool) (api.ExpressionEmission, error) {
	logical, err := owner.StorageType(context, source, sourceType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	physical, err := owner.MemoryStorageType(context, source, sourceType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	input, output := logical, physical
	if !toMemory {
		input, output = physical, logical
	}
	parameter, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	factory := context.Factory()
	layout, _, fields, err := memorymarker.RecordBindingLayout(context, owner.children, source, sourceType, toMemory)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments := []api.ExpressionEmission{layout}
	for index := range structure.NumFields() {
		field := structure.Field(index)
		name, err := structconstruction.FieldName(context.Names(), field, index)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		fieldType := field.Type()
		inputType, err := owner.StorageType(context, source, fieldType)
		if !toMemory && err == nil {
			inputType, err = owner.MemoryStorageType(context, source, fieldType)
		}
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		resultType, err := owner.MemoryStorageType(context, source, fieldType)
		if !toMemory && err == nil {
			resultType, err = owner.StorageType(context, source, fieldType)
		}
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		property := factory.PropertyAccessExpression(factory.Identifier(parameter), nil, factory.Identifier(name), tsgo.NodeFlagsNone)
		base, err := pointermarker.Operation(context, tsoniccore.SymbolAddressOf, []api.TypeEmission{inputType}, []api.ExpressionEmission{api.DirectExpression(property)})
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !memoryStorageDiffers(context, fieldType) {
			binding, err := pointermarker.Operation(context, tsoniccore.SymbolBindMemoryField, nil, []api.ExpressionEmission{fields[index], base})
			if err != nil {
				return api.ExpressionEmission{}, err
			}
			arguments = append(arguments, binding)
			continue
		}
		read, err := owner.memoryRecordField(context, source, fieldType, api.DirectExpression(property), toMemory)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		setter, err := context.Names().Temporary(api.TemporaryConversionOperand)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		write, err := owner.memoryRecordField(context, source, fieldType, api.DirectExpression(factory.Identifier(setter)), !toMemory)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if toMemory {
			switch fieldType.Underlying().(type) {
			case *types.Array, *types.Struct:
				write, err = owner.assignMemoryRecordField(context, source, fieldType, property, write)
				if err != nil {
					return api.ExpressionEmission{}, err
				}
			}
		}
		assignment := factory.BinaryExpression(nil, property, nil, factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken), write.Value())
		readView := factory.ArrowFunction(nil, nil, nil, resultType.Value(), factory.EqualsGreaterThanToken(),
			factory.Block(append(read.Before(), factory.ReturnStatement(read.Value())), true))
		writeView := factory.ArrowFunction(nil, nil, []tsgo.ParameterDeclaration{
			factory.ParameterDeclaration(nil, nil, factory.Identifier(setter), nil, resultType.Value(), nil),
		}, factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword), factory.EqualsGreaterThanToken(),
			factory.Block(append(write.Before(), factory.ExpressionStatement(assignment)), true))
		pointer, err := pointermarker.Operation(context, tsoniccore.SymbolViewPointer, []api.TypeEmission{inputType, resultType}, []api.ExpressionEmission{
			base, api.DirectExpression(readView, read.Requests()...), api.DirectExpression(writeView, write.Requests()...),
		})
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		binding, err := pointermarker.Operation(context, tsoniccore.SymbolBindMemoryField, nil, []api.ExpressionEmission{fields[index], pointer})
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		arguments = append(arguments, binding)
	}
	view, err := pointermarker.Operation(context, tsoniccore.SymbolBindMemoryRecord, nil, arguments)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return descriptorConversion(context, parameter, input, output, value, view)
}

func (owner Owner) assignMemoryRecordField(context api.Context, source ast.Node, fieldType types.Type, property tsgo.Expression, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	if context.TypesSizes().Sizeof(fieldType) == 0 {
		return api.NewExpressionEmission(append(value.Before(), context.Factory().ExpressionStatement(value.Value())), property, value.Requests())
	}
	current, err := owner.FromStorage(context, source, fieldType, api.DirectExpression(property))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	incoming, err := owner.FromStorage(context, source, fieldType, value)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	updated, err := context.StableAssignments().AssignStable(context, source, fieldType, current.Value(), incoming)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(current.Before(), updated.Before()...)
	before = append(before, context.Factory().ExpressionStatement(updated.Value()))
	return api.NewExpressionEmission(before, property, api.CombineRequests(current.Requests(), updated.Requests()))
}

func (owner Owner) memoryRecordField(context api.Context, source ast.Node, fieldType types.Type, value api.ExpressionEmission, toMemory bool) (api.ExpressionEmission, error) {
	if toMemory {
		logical, err := owner.FromStorage(context, source, fieldType, value)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return owner.ToMemoryStorage(context, source, fieldType, logical)
	}
	logical, err := owner.FromMemoryStorage(context, source, fieldType, value)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return owner.ToStorage(context, source, fieldType, logical)
}
