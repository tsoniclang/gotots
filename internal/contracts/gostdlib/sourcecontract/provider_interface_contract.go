package sourcecontract

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

type ProviderInterfaceMethodSelection struct {
	Method      *types.Func
	Certificate gostdlib.ProviderInterfaceMethod
}

func SelectProviderInterfaceMethods(
	certificate gostdlib.ProviderCallableProfileInterface,
	contract *types.Interface,
) ([]ProviderInterfaceMethodSelection, bool, error) {
	if certificate.SourceIdentity() == "" || certificate.Export() == "" ||
		contract == nil || !contract.Complete().IsMethodSet() {
		return nil, false, &ProtocolError{
			Field:  "providerInterface",
			Reason: "capability contract evidence is incomplete",
		}
	}
	contract = contract.Complete()
	provider := certificate.ProviderInterface()
	if provider.Mode() != gostdlib.ProviderInterfaceModeBridge {
		return nil, false, nil
	}
	protocol, synthetic := certificate.Protocol()
	result := make(
		[]ProviderInterfaceMethodSelection,
		0,
		contract.NumMethods(),
	)
	for index := range contract.NumMethods() {
		method := contract.Method(index)
		identity := ""
		if synthetic {
			selected, ok := gostdlib.ProviderProtocolMethod(protocol, method.Name())
			if !ok {
				return nil, false, nil
			}
			var err error
			identity, _, err = gostdlib.ProviderProtocolMethodSource(
				certificate.SourceIdentity(),
				selected,
			)
			if err != nil {
				return nil, false, err
			}
		} else {
			var err error
			identity, _, err = ProviderInterfaceMethod(method)
			if err != nil {
				return nil, false, err
			}
		}
		selected, ok := provider.Method(identity)
		methodSignature, signatureOK := environmentcontract.MethodSignature(method)
		if !ok || !signatureOK ||
			selected.ContractSignature() !=
				environmentcontract.StableTypeString(methodSignature) {
			return nil, false, nil
		}
		result = append(result, ProviderInterfaceMethodSelection{
			Method:      method,
			Certificate: selected,
		})
	}
	return result, true, nil
}
