package sourcecontract

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func ObjectIdentity(object types.Object) (string, error) {
	if object == types.Universe.Lookup("error") {
		return gostdlib.LanguageErrorInterfaceIdentity, nil
	}
	contract, err := environmentcontract.Describe(object)
	if err != nil {
		return "", err
	}
	return contract.Identity(), nil
}
