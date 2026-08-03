package sourcecontract

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func ProviderInterfaceMethod(
	method *types.Func,
) (string, string, error) {
	if method != nil && method.Origin() == languageErrorMethod() {
		return gostdlib.LanguageErrorMethodIdentity, "func() string", nil
	}
	contract, err := environmentcontract.Describe(method)
	if err != nil {
		return "", "", err
	}
	return contract.Identity(), contract.Signature(), nil
}

func languageErrorMethod() *types.Func {
	typeName, _ := types.Universe.Lookup("error").(*types.TypeName)
	named, _ := types.Unalias(typeName.Type()).(*types.Named)
	contract, _ := named.Underlying().(*types.Interface)
	return contract.Complete().Method(0).Origin()
}
