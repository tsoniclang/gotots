package memory

import (
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	rawpointermarker "github.com/tsoniclang/gotots/internal/emit/marker/rawpointer"
	descriptorruntime "github.com/tsoniclang/gotots/internal/emit/runtime/memorydescriptor"
	descriptorvalue "github.com/tsoniclang/gotots/internal/emit/value/memorydescriptor"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func DescriptorLayout(context api.Context, model descriptorvalue.Model) (api.ExpressionEmission, api.TypeEmission, error) {
	storage, err := model.StorageType(context)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	word, err := model.WordType(context)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	data, err := rawpointermarker.Type(context, true)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	abi, err := DataLayout(context)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	number := func(value int64) api.ExpressionEmission {
		return api.DirectExpression(context.Factory().NumericLiteral(strconv.FormatInt(value, 10), tsgo.TokenFlagsNone))
	}
	alignment := context.TypesSizes().Alignof(types.Typ[types.Int])
	child := func(target api.TypeEmission) (api.ExpressionEmission, error) {
		return pointermarker.Operation(context, tsoniccore.SymbolMemoryLayout, []api.TypeEmission{target},
			[]api.ExpressionEmission{abi, number(model.WordBytes()), number(alignment), number(model.WordBytes())})
	}
	dataLayout, err := child(data)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	wordLayout, err := child(word)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	dataField, err := selectedFieldLayout(context, storage, descriptorruntime.DataMember, 0, alignment, dataLayout)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	lengthField, err := selectedFieldLayout(context, storage, descriptorruntime.LengthMember, model.WordBytes(), alignment, wordLayout)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	size := context.TypesSizes().Sizeof(model.SourceType())
	arguments := []api.ExpressionEmission{abi, number(size), number(alignment), number(size), dataField, lengthField}
	if model.IsSlice() {
		capacityField, err := selectedFieldLayout(context, storage, descriptorruntime.CapacityMember, 2*model.WordBytes(), alignment, wordLayout)
		if err != nil {
			return api.ExpressionEmission{}, api.TypeEmission{}, err
		}
		arguments = append(arguments, capacityField)
	}
	layout, err := pointermarker.Operation(context, tsoniccore.SymbolMemoryLayout, []api.TypeEmission{storage}, arguments)
	return layout, storage, err
}
