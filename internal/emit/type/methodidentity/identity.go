package methodidentity

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
)

func Signature(method *types.Func) (*types.Signature, bool) {
	if method == nil {
		return nil, false
	}
	source, ok := method.Type().(*types.Signature)
	if !ok {
		return nil, false
	}
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		source.Params(),
		source.Results(),
		source.Variadic(),
	), true
}

func BuildKey(
	method *types.Func,
	namedObjectIdentity typeidentity.NamedObjectIdentity,
) (string, error) {
	if method == nil {
		return "", &api.NameError{
			Reason: "interface method identity is nil",
		}
	}
	signature, ok := Signature(method)
	if !ok {
		return "", &api.NameError{
			Reason: "interface method has no receiver-free signature",
		}
	}
	origin := method.Origin()
	parameters := api.GenericDeclarationParameters(origin)
	if method != origin {
		parameters = nil
	}
	var signatureKey string
	var err error
	if len(parameters) == 0 {
		signatureKey, err = typeidentity.BuildKey(
			signature,
			namedObjectIdentity,
		)
	} else {
		indices := make(map[*types.TypeParam]int, len(parameters))
		for index, parameter := range parameters {
			indices[parameter] = index
		}
		signatureKey, err = typeidentity.BuildParameterizedKey(
			signature,
			namedObjectIdentity,
			func(parameter *types.TypeParam) (string, error) {
				index, found := indices[parameter]
				if !found {
					return "", &api.NameError{
						Reason: "interface method type parameter has no declaration identity",
					}
				}
				return "receiver|" + strconv.Itoa(index), nil
			},
		)
	}
	if err != nil {
		return "", err
	}
	identity := origin.Name()
	if !origin.Exported() {
		if origin.Pkg() == nil {
			return "", &api.NameError{
				Name:   origin.Name(),
				Reason: "unexported interface method has no package identity",
			}
		}
		identity = origin.Pkg().Path() + "\x00" + identity
	}
	digest := sha256.Sum256(
		[]byte(identity + "\x00" + signatureKey),
	)
	return hex.EncodeToString(digest[:]), nil
}

func Equivalent(left *types.Func, right *types.Func) bool {
	leftSignature, leftOK := Signature(left)
	rightSignature, rightOK := Signature(right)
	if !leftOK || !rightOK ||
		left.Name() != right.Name() ||
		left.Exported() != right.Exported() ||
		!types.Identical(leftSignature, rightSignature) {
		return false
	}
	if left.Exported() {
		return true
	}
	return left.Pkg() != nil &&
		right.Pkg() != nil &&
		left.Pkg().Path() == right.Pkg().Path()
}
