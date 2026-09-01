package call

import (
	"go/ast"
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/expression/call/interfaceoperation"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitInterfaceMethod(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	selector *ast.SelectorExpr,
	method *types.Func,
	selection *types.Selection,
	signature *types.Signature,
	discarded bool,
	_ bool,
) (api.ExpressionEmission, error) {
	providerInterface, providerOwned, err :=
		context.Names().ProviderInterface(selection.Recv())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	nativeProvider := providerOwned && providerInterface.Mode() ==
		gostdlib.ProviderInterfaceModeSealedNative
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleReceiverValue).
			WithExpectedType(selection.Recv()),
		selector.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments, argumentBefore, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
		nativeProvider,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if nativeProvider {
		var providerBefore []tsgo.Statement
		var providerRequests []api.RootRequest
		arguments, providerBefore, providerRequests, err =
			providerboundary.ToProviderArguments(
				context,
				children,
				signature.Params(),
				arguments,
			)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		argumentBefore = append(argumentBefore, providerBefore...)
		argumentRequests = api.CombineRequests(
			argumentRequests,
			providerRequests,
		)
	}
	target, err := interfaceoperation.Apply(
		context,
		children,
		selector.X,
		selection.Recv(),
		receiver,
		method,
		arguments,
		argumentBefore,
		argumentRequests,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if nativeProvider {
		contract, contractErr := environmentcontract.Describe(method.Origin())
		if contractErr != nil {
			return api.ExpressionEmission{}, contractErr
		}
		certificate, found := providerInterface.Method(contract.Identity())
		member, memberErr := context.Names().InterfaceMethodName(method)
		if memberErr != nil {
			return api.ExpressionEmission{}, memberErr
		}
		if !found ||
			certificate.Kind() != gostdlib.ProviderInterfaceMethodCallable ||
			certificate.Member() != member ||
			certificate.Effect() != gostdlib.EffectSynchronous {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "sealed provider interface method certificate is invalid",
			}
		}
		if discarded {
			return target, nil
		}
		return providerboundary.FromProviderResults(
			context,
			children,
			nil,
			"",
			signature.Results(),
			target,
		)
	}
	return target, nil
}
