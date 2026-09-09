package naming

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (names *File) lifetimeReference(object types.Object, kind targetBindingKind, phase api.ImportPhase) (api.NameReference, bool, error) {
	function, callable := object.(*types.Func)
	if !callable || phase != api.ImportPhaseValue ||
		kind != targetBindingEnvironment && kind != targetBindingProvider && kind != targetBindingMissingProvider {
		return api.NameReference{}, false, nil
	}
	contract, err := environmentcontract.Describe(function)
	if err != nil {
		return api.NameReference{}, true, err
	}
	if contract.Identity() != "runtime|kind=4|receiver=|name=KeepAlive" {
		return api.NameReference{}, false, nil
	}
	signature, valid := function.Type().(*types.Signature)
	if !valid || signature.Recv() != nil || signature.TypeParams().Len() != 0 ||
		signature.Params().Len() != 1 || signature.Results().Len() != 0 || signature.Variadic() {
		return api.NameReference{}, true, &api.NameError{Name: contract.Identity(), Reason: "KeepAlive requires its exact selected Go callable contract"}
	}
	argument, valid := signature.Params().At(0).Type().Underlying().(*types.Interface)
	if !valid || !argument.Empty() {
		return api.NameReference{}, true, &api.NameError{Name: contract.Identity(), Reason: "KeepAlive requires an empty-interface argument"}
	}
	if err := names.ObserveEnvironmentImplementation(function, environmentcontract.UseDemandCallable, environmentcontract.RouteGeneratedFacet); err != nil {
		return api.NameReference{}, true, err
	}
	reference, err := names.Runtime(api.RuntimeKeepAlive, phase)
	return reference, true, err
}
