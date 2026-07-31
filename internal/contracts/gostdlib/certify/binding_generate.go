package certify

import (
	"fmt"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func bindingDocument(
	evidence goObject,
	export string,
	member string,
	access gostdlib.AccessKind,
	fingerprint string,
	owners []string,
) (gostdlib.BindingDocument, error) {
	if fingerprint == "" {
		return gostdlib.BindingDocument{}, certifyError(
			"build binding",
			evidence.contract.Identity(),
			"target fingerprint is absent",
		)
	}
	if len(owners) != 1 {
		return gostdlib.BindingDocument{}, certifyError(
			"build binding",
			evidence.contract.Identity(),
			fmt.Sprintf("target has %d implementation owners, want one", len(owners)),
		)
	}
	kind, err := bindingKind(evidence.contract.Kind())
	if err != nil {
		return gostdlib.BindingDocument{}, err
	}
	representation := gostdlib.RepresentationInvalid
	if kind == gostdlib.BindingType {
		representation = gostdlib.RepresentationDirect
	}
	return gostdlib.BindingDocument{
		Identity:            evidence.contract.Identity(),
		Kind:                kind,
		Access:              access,
		Representation:      representation,
		Export:              export,
		Member:              member,
		SourceSignature:     evidence.contract.Signature(),
		SourceValue:         evidence.contract.Value(),
		SourceLocation:      evidence.location,
		ImplementationOwner: owners[0],
		TargetFingerprint:   fingerprint,
	}, nil
}

func bindingKind(source environmentcontract.ObjectKind) (gostdlib.BindingKind, error) {
	switch source {
	case environmentcontract.ObjectConstant:
		return gostdlib.BindingConstant, nil
	case environmentcontract.ObjectType:
		return gostdlib.BindingType, nil
	case environmentcontract.ObjectVariable:
		return gostdlib.BindingVariable, nil
	case environmentcontract.ObjectFunction:
		return gostdlib.BindingFunction, nil
	case environmentcontract.ObjectBuiltin:
		return gostdlib.BindingBuiltin, nil
	default:
		return gostdlib.BindingInvalid, certifyError(
			"build binding",
			fmt.Sprint(source),
			"Go object kind is unsupported",
		)
	}
}
