package memory

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	descriptorvalue "github.com/tsoniclang/gotots/internal/emit/value/memorydescriptor"
	"github.com/tsoniclang/gotots/internal/emit/value/structconstruction"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) MemoryStorageType(context api.Context, source ast.Node, sourceType types.Type) (api.TypeEmission, error) {
	if model, selected := descriptorvalue.Resolve(context, sourceType); selected {
		return model.StorageType(context)
	}
	if array, selected := arrayvalue.Resolve(context, sourceType); selected {
		return array.MemoryStorageType(context, source)
	}
	if structure, selected := sourceType.Underlying().(*types.Struct); selected && memoryFieldsDiffer(context, structure) {
		var members []tsgo.TypeElement
		var requests []api.RootRequest
		for index := range structure.NumFields() {
			field := structure.Field(index)
			name, err := structconstruction.FieldName(context.Names(), field, index)
			if err != nil {
				return api.TypeEmission{}, err
			}
			member, err := owner.MemoryStorageType(context, source, field.Type())
			if err != nil {
				return api.TypeEmission{}, err
			}
			members = append(members, context.Factory().PropertySignatureDeclaration(nil, context.Factory().Identifier(name), nil, member.Value(), context.Factory().OmittedExpression()))
			requests = append(requests, member.Requests()...)
		}
		return api.DirectType(context.Factory().TypeLiteralNode(members), requests...), nil
	}
	return owner.StorageType(context, source, sourceType)
}

func (owner Owner) ToMemoryStorage(context api.Context, source ast.Node, sourceType types.Type, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	if model, selected := descriptorvalue.Resolve(context, sourceType); selected {
		if model.IsSlice() {
			return owner.sliceToDescriptor(context, source, model, value)
		}
		return owner.stringToDescriptor(context, source, model, value)
	}
	if array, selected := arrayvalue.Resolve(context, sourceType); selected {
		return owner.arrayToMemory(context, source, array, value)
	}
	stored, err := owner.ToStorage(context, source, sourceType, value)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if structure, selected := sourceType.Underlying().(*types.Struct); selected && memoryFieldsDiffer(context, structure) {
		return owner.memoryRecord(context, source, sourceType, structure, stored, true)
	}
	return stored, nil
}

func (owner Owner) FromMemoryStorage(context api.Context, source ast.Node, sourceType types.Type, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	if model, selected := descriptorvalue.Resolve(context, sourceType); selected {
		if model.IsSlice() {
			return owner.sliceFromDescriptor(context, source, model, value)
		}
		return owner.stringFromDescriptor(context, source, model, value)
	}
	if array, selected := arrayvalue.Resolve(context, sourceType); selected {
		return owner.arrayFromMemory(context, source, array, value)
	}
	stored := value
	if structure, selected := sourceType.Underlying().(*types.Struct); selected && memoryFieldsDiffer(context, structure) {
		var err error
		stored, err = owner.memoryRecord(context, source, sourceType, structure, stored, false)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	return owner.FromStorage(context, source, sourceType, stored)
}

func memoryFieldsDiffer(context api.Context, structure *types.Struct) bool {
	for index := range structure.NumFields() {
		if memoryStorageDiffers(context, structure.Field(index).Type()) {
			return true
		}
	}
	return false
}

func memoryStorageDiffers(context api.Context, sourceType types.Type) bool {
	if _, selected := descriptorvalue.Resolve(context, sourceType); selected {
		return true
	}
	switch underlying := sourceType.Underlying().(type) {
	case *types.Array:
		return true
	case *types.Struct:
		return memoryFieldsDiffer(context, underlying)
	default:
		return false
	}
}
