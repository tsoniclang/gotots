package certify

import (
	"fmt"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func verifyExportSourceCallableShape(
	project *tsgo.ProjectInspection,
	evidence goObject,
	target tsgo.ProjectExport,
) error {
	signature, ok := evidence.object.Type().(*types.Signature)
	if !ok {
		return certifyError(
			"verify source callable shape",
			evidence.contract.Identity(),
			"selected Go function signature is absent",
		)
	}
	actual, err := project.CallableParameterCount(target)
	if err != nil {
		return err
	}
	actualTypes, err := project.CallableTypeParameterCount(target)
	if err != nil {
		return err
	}
	return verifySourceCallableShape(
		evidence.contract.Identity(),
		signature,
		gostdlib.AccessExport,
		actual,
		actualTypes,
	)
}

func verifyMethodSourceCallableShape(
	project *tsgo.ProjectInspection,
	evidence goObject,
	target tsgo.ProjectMember,
	access gostdlib.AccessKind,
) error {
	signature, ok := evidence.object.Type().(*types.Signature)
	if !ok {
		return certifyError(
			"verify source callable shape",
			evidence.contract.Identity(),
			"selected Go method signature is absent",
		)
	}
	actual, err := project.CallableParameterCount(target)
	if err != nil {
		return err
	}
	actualTypes, err := project.CallableTypeParameterCount(target)
	if err != nil {
		return err
	}
	return verifySourceCallableShape(
		evidence.contract.Identity(),
		signature,
		access,
		actual,
		actualTypes,
	)
}

func verifySourceCallableShape(
	identity string,
	signature *types.Signature,
	access gostdlib.AccessKind,
	actualValues int,
	actualTypes int,
) error {
	expectedValues, err := sourceCallableParameterCount(signature, access)
	if err != nil {
		return certifyError("verify source callable shape", identity, err.Error())
	}
	if actualValues != expectedValues {
		return certifyError(
			"verify source callable shape",
			identity,
			fmt.Sprintf(
				"target has %d value parameters, selected Go shape requires %d",
				actualValues,
				expectedValues,
			),
		)
	}
	expectedTypes := sourceCallableTypeParameterCount(signature)
	if actualTypes != expectedTypes {
		return certifyError(
			"verify source callable shape",
			identity,
			fmt.Sprintf(
				"target has %d type parameters, selected Go shape requires %d",
				actualTypes,
				expectedTypes,
			),
		)
	}
	return nil
}

func sourceCallableTypeParameterCount(signature *types.Signature) int {
	if signature == nil {
		return 0
	}
	count := 0
	if signature.RecvTypeParams() != nil {
		count += signature.RecvTypeParams().Len()
	}
	if signature.TypeParams() != nil {
		count += signature.TypeParams().Len()
	}
	return count
}

func sourceCallableParameterCount(
	signature *types.Signature,
	access gostdlib.AccessKind,
) (int, error) {
	if signature == nil {
		return 0, fmt.Errorf("selected Go signature is absent")
	}
	parameterCount := 0
	if signature.Params() != nil {
		parameterCount = signature.Params().Len()
	}
	receiver := signature.Recv()
	switch access {
	case gostdlib.AccessExport:
		if receiver != nil {
			return 0, fmt.Errorf("package export unexpectedly has a receiver")
		}
		return parameterCount, nil
	case gostdlib.AccessInstanceMethod:
		if receiver == nil {
			return 0, fmt.Errorf("instance method receiver is absent")
		}
		if _, pointer := receiver.Type().(*types.Pointer); pointer {
			return 0, fmt.Errorf("pointer receiver cannot use an instance operation")
		}
		return parameterCount, nil
	case gostdlib.AccessStaticMethod:
		if receiver == nil {
			return 0, fmt.Errorf("static method receiver is absent")
		}
		if _, pointer := receiver.Type().(*types.Pointer); !pointer {
			return 0, fmt.Errorf("value receiver cannot use a static operation")
		}
		return parameterCount + 1, nil
	default:
		return 0, fmt.Errorf("callable access %q is unsupported", access)
	}
}
