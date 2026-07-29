package methodidentity

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"

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
	signature, ok := Signature(method)
	if !ok {
		return "", &api.NameError{
			Reason: "interface method has no receiver-free signature",
		}
	}
	signatureKey, err := typeidentity.BuildKey(
		signature,
		namedObjectIdentity,
	)
	if err != nil {
		return "", err
	}
	identity := method.Name()
	if !method.Exported() {
		if method.Pkg() == nil {
			return "", &api.NameError{
				Name:   method.Name(),
				Reason: "unexported interface method has no package identity",
			}
		}
		identity = method.Pkg().Path() + "\x00" + identity
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
