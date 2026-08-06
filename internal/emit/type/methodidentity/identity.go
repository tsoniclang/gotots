package methodidentity

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"
	"strconv"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
)

func Signature(method *types.Func) (*types.Signature, bool) {
	return environmentcontract.MethodSignature(method)
}

func BuildKey(
	method *types.Func,
	namedObjectIdentity typeidentity.NamedObjectIdentity,
) (string, error) {
	components, err := buildContractComponents(method)
	if err != nil {
		return "", err
	}
	var signatureKey string
	if len(components.parameters) == 0 {
		signatureKey, err = typeidentity.BuildKey(
			components.signature,
			namedObjectIdentity,
		)
	} else {
		signatureKey, err = typeidentity.BuildParameterizedKey(
			components.signature,
			namedObjectIdentity,
			parameterIdentity(components.parameters),
		)
	}
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(
		[]byte(components.identity + "\x00" + signatureKey),
	)
	return hex.EncodeToString(digest[:]), nil
}

func ContractDescriptor(
	method *types.Func,
	namedObjectIdentity typeidentity.NamedObjectIdentity,
) (string, error) {
	components, err := buildContractComponents(method)
	if err != nil {
		return "", err
	}
	var signatureDescriptor string
	if len(components.parameters) == 0 {
		signatureDescriptor, err = typeidentity.BuildDescriptor(
			components.signature,
			namedObjectIdentity,
		)
	} else {
		signatureDescriptor, err = typeidentity.BuildParameterizedDescriptor(
			components.signature,
			namedObjectIdentity,
			parameterIdentity(components.parameters),
		)
	}
	if err != nil {
		return "", err
	}
	return strconv.Itoa(len(components.identity)) + ":" +
		components.identity +
		signatureDescriptor, nil
}

type contractComponents struct {
	identity   string
	signature  *types.Signature
	parameters []*types.TypeParam
}

func buildContractComponents(method *types.Func) (contractComponents, error) {
	if method == nil {
		return contractComponents{}, &api.NameError{
			Reason: "interface method identity is nil",
		}
	}
	signature, ok := Signature(method)
	if !ok {
		return contractComponents{}, &api.NameError{
			Reason: "interface method has no receiver-free signature",
		}
	}
	origin := method.Origin()
	parameters := identityParameters(origin)
	if method != origin {
		parameters = nil
	}
	identity := origin.Name()
	if !origin.Exported() {
		if origin.Pkg() == nil {
			return contractComponents{}, &api.NameError{
				Name:   origin.Name(),
				Reason: "unexported interface method has no package identity",
			}
		}
		identity = origin.Pkg().Path() + "\x00" + identity
	}
	return contractComponents{
		identity:   identity,
		signature:  signature,
		parameters: parameters,
	}, nil
}

func parameterIdentity(
	parameters []*types.TypeParam,
) typeidentity.TypeParameterIdentity {
	indices := make(map[*types.TypeParam]int, len(parameters))
	for index, parameter := range parameters {
		indices[parameter] = index
	}
	return func(parameter *types.TypeParam) (string, error) {
		index, found := indices[parameter]
		if !found {
			return "", &api.NameError{
				Reason: "interface method type parameter has no declaration identity",
			}
		}
		return "receiver|" + strconv.Itoa(index), nil
	}
}

func identityParameters(method *types.Func) []*types.TypeParam {
	parameters := api.GenericDeclarationParameters(method)
	if len(parameters) != 0 {
		return parameters
	}
	signature, _ := method.Type().(*types.Signature)
	if signature == nil || signature.Recv() == nil {
		return nil
	}
	receiver := signature.Recv().Type()
	if pointer, ok := types.Unalias(receiver).(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	named, ok := types.Unalias(receiver).(*types.Named)
	if !ok || named.Origin() == nil {
		return nil
	}
	parameters = make(
		[]*types.TypeParam,
		0,
		named.Origin().TypeParams().Len(),
	)
	for index := range named.Origin().TypeParams().Len() {
		parameters = append(
			parameters,
			named.Origin().TypeParams().At(index),
		)
	}
	return parameters
}
