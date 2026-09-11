package reflectiontype

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/goabi"
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type rawContractServices struct {
	api.Names
	api.Values
	api.ChildEmitter
	pointerType tsgo.TypeNode
}

func (*rawContractServices) Temporary(api.TemporaryKind) (string, error) {
	return "pointerValue", nil
}

func (*rawContractServices) SourceDataLayout(goabi.Layout) (api.NameReference, error) {
	return api.NewNameReference("abi")
}

func (*rawContractServices) TsonicCore(symbol tsoniccore.Symbol) (api.NameReference, error) {
	contract, err := tsoniccore.Resolve(symbol)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(contract.Export())
}

func (*rawContractServices) RequiresStorageProjection(api.Context, types.Type) (bool, error) {
	return false, nil
}

func (services *rawContractServices) MemoryStorageType(context api.Context, source ast.Node, value types.Type) (api.TypeEmission, error) {
	return services.RepresentedType(context, source, value)
}

func (services *rawContractServices) RepresentedType(context api.Context, source ast.Node, value types.Type) (api.TypeEmission, error) {
	element := api.DirectType(context.Factory().TypeReferenceNode(context.Factory().Identifier("uint32"), nil))
	if _, pointer := value.(*types.Pointer); !pointer {
		return element, nil
	}
	emitted, err := pointermarker.Type(context, element, true)
	services.pointerType = emitted.Value()
	return emitted, err
}

func TestRawCallbackPublishesExactPointeeType(test *testing.T) {
	services := &rawContractServices{}
	context, err := api.NewContext(api.RoleParameterType, token.NewFileSet(),
		types.NewPackage("example.com/raw", "raw"), &types.Info{}, types.SizesFor("gc", "amd64"),
		api.MemoryByteOrderLittleEndian, tsgo.NewFactory(), services, services,
		api.IntegerRepresentationNumber, api.EvaluationOrderDirect)
	if err != nil {
		test.Fatal(err)
	}
	emitted, _, err := pointerRawOperation(context, services, types.Typ[types.Uint32])
	if err != nil {
		test.Fatal(err)
	}
	callback, ok := emitted.(tsgo.ArrowFunction)
	if !ok || len(callback.Parameters()) != 1 {
		test.Fatal("raw callback must retain one source pointer parameter")
	}
	if services.pointerType == nil || callback.Parameters()[0].Type() != services.pointerType {
		test.Fatal("raw callback lost the exact pointer type selected by its existing type owner")
	}
}
