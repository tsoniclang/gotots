package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/generic/semanticname"
	"github.com/tsoniclang/gotots/internal/emit/type/methodidentity"
)

func (r *Registry) GeneratedGenericKernelTransport(
	owner *types.Func,
) (api.GeneratedRepresentationTransport, error) {
	if r == nil || owner == nil || owner.Origin() != owner ||
		len(api.GenericDeclarationParameters(owner)) == 0 {
		return api.GeneratedRepresentationTransport{}, &api.NameError{
			Reason: "generated generic kernel owner is invalid",
		}
	}
	binding, ok := r.byObject[owner]
	if !ok || !binding.sourceOwned() || binding.name == "" || binding.sourcePath == "" {
		return api.GeneratedRepresentationTransport{}, &api.NameError{
			Name:   owner.Name(),
			Reason: "generated generic kernel has no source declaration",
		}
	}
	signature, ok := owner.Type().(*types.Signature)
	if !ok {
		return api.GeneratedRepresentationTransport{}, &api.NameError{
			Name:   owner.Name(),
			Reason: "generated generic kernel signature is invalid",
		}
	}
	if signature.Recv() == nil ||
		sourceMethodTargetKind(owner) == api.MethodTargetSourceFunction {
		return api.NewGeneratedRepresentationTransport(
			api.GeneratedRepresentationTransportFunctionKernel,
			binding.sourcePath,
			binding.name+api.GenericKernelSuffix,
			"",
		)
	}
	receiver := api.MethodReceiverTypeName(owner)
	receiverBinding, ok := r.byObject[receiver]
	if !ok || !receiverBinding.sourceOwned() || receiverBinding.name == "" ||
		receiverBinding.sourcePath == "" {
		return api.GeneratedRepresentationTransport{}, &api.NameError{
			Name:   owner.Name(),
			Reason: "generated generic member kernel has no receiver declaration",
		}
	}
	member, err := r.interfaceMethodName(owner)
	if err != nil {
		return api.GeneratedRepresentationTransport{}, err
	}
	return api.NewGeneratedRepresentationTransport(
		api.GeneratedRepresentationTransportMemberKernel,
		receiverBinding.sourcePath,
		receiverBinding.name,
		member+api.GenericKernelSuffix,
	)
}

func (r *Registry) interfaceMethodName(method *types.Func) (string, error) {
	if method == nil {
		return "", &api.NameError{Reason: "interface method is nil"}
	}
	if _, ok := methodidentity.Signature(method); !ok {
		return "", &api.NameError{
			Name:   method.Name(),
			Reason: "interface method signature is invalid",
		}
	}
	if method.Exported() {
		return portableIdentifier(method.Name()), nil
	}
	qualifier, err := r.semanticPackageToken(method.Pkg())
	if err != nil {
		return "", err
	}
	return "$go$private$" + qualifier + "$" +
		semanticname.Identifier(method.Name()), nil
}
