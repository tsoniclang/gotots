package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/generic/semanticname"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
)

func (n *File) semanticGeneratedTypeName(
	prefix string,
	sourceType types.Type,
) (string, error) {
	return semanticGeneratedTypeName(
		prefix,
		sourceType,
		n.generatedNamedObjectIdentity,
	)
}

func semanticGeneratedTypeName(
	prefix string,
	sourceType types.Type,
	localIdentity semanticname.LocalNamedIdentity,
) (string, error) {
	var (
		semanticType string
		err          error
	)
	if len(typeidentity.LocalComponents(sourceType)) == 0 {
		semanticType, err = semanticname.Type(sourceType)
	} else {
		semanticType, err = semanticname.TypeWithLocalIdentity(
			sourceType,
			localIdentity,
		)
	}
	if err != nil {
		return "", err
	}
	return prefix + semanticType, nil
}

func (n *File) semanticGeneratedMethodName(
	prefix string,
	method *types.Func,
	signature *types.Signature,
) (string, error) {
	if method == nil || signature == nil {
		return "", &api.NameError{
			Reason: "semantic generated method name is invalid",
		}
	}
	contract, err := n.semanticGeneratedTypeName("", signature)
	if err != nil {
		return "", err
	}
	identity := semanticname.Identifier(
		types.Id(method.Origin().Pkg(), method.Origin().Name()),
	)
	return prefix + identity + "$" + contract, nil
}
