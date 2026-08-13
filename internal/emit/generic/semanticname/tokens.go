package semanticname

import (
	"go/types"
	"strings"
)

type LocalNamedIdentity func(*types.TypeName) (string, error)
type NamedTypeToken func(*types.TypeName) (string, error)
type PackageToken func(*types.Package) (string, error)

type typeIdentityResolver struct {
	localIdentity LocalNamedIdentity
	namedToken    NamedTypeToken
	packageToken  PackageToken
}

func ConcretizationSuffixWithIdentityTokens(
	arguments []types.Type,
	synchronous bool,
	namedToken NamedTypeToken,
	packageToken PackageToken,
) (string, error) {
	if namedToken == nil || packageToken == nil {
		return "", invalid("generic concretization identity-token owner is nil")
	}
	return concretizationSuffix(
		arguments,
		synchronous,
		typeIdentityResolver{
			namedToken:   namedToken,
			packageToken: packageToken,
		},
	)
}

func concretizationSuffix(
	arguments []types.Type,
	synchronous bool,
	identity typeIdentityResolver,
) (string, error) {
	if len(arguments) == 0 {
		return "", invalid("generic concretization has no type arguments")
	}
	parts := make([]string, 0, len(arguments)+1)
	for _, argument := range arguments {
		part, err := typeWithIdentity(argument, identity)
		if err != nil {
			return "", err
		}
		parts = append(parts, part)
	}
	if synchronous {
		parts = append(parts, "synchronous")
	}
	return "$" + strings.Join(parts, "$"), nil
}

func OperationNameWithIdentityTokens(
	operation string,
	method *types.Func,
	signature *types.Signature,
	namedToken NamedTypeToken,
	packageToken PackageToken,
) (string, error) {
	if namedToken == nil || packageToken == nil {
		return "", invalid("generic operation identity-token owner is nil")
	}
	return operationName(
		operation,
		method,
		signature,
		typeIdentityResolver{
			namedToken:   namedToken,
			packageToken: packageToken,
		},
	)
}

func operationName(
	operation string,
	method *types.Func,
	signature *types.Signature,
	identity typeIdentityResolver,
) (string, error) {
	if operation == "" || signature == nil || signature.Recv() != nil {
		return "", invalid("generic operation contract is invalid")
	}
	contract, err := signatureTokenWithPending(
		signature,
		make(map[types.Type]struct{}),
		make(map[*types.TypeParam]int),
		identity,
	)
	if err != nil {
		return "", err
	}
	operation, err = operationIdentifier(operation)
	if err != nil {
		return "", err
	}
	name := "$go$" + operation
	if method != nil {
		method = method.Origin()
		if _, ok := method.Type().(*types.Signature); !ok {
			return "", invalid("generic operation method is invalid")
		}
		methodName, nameErr := packageObjectName(
			method.Pkg(),
			method.Name(),
			identity,
		)
		if nameErr != nil {
			return "", nameErr
		}
		name += "$" + methodName
	}
	return name + "$" + contract, nil
}

func packageObjectName(
	sourcePackage *types.Package,
	name string,
	identity typeIdentityResolver,
) (string, error) {
	if sourcePackage == nil {
		return identifier(name), nil
	}
	if identity.packageToken == nil {
		return identifier(types.Id(sourcePackage, name)), nil
	}
	packageName, err := identity.packageToken(sourcePackage)
	if err != nil {
		return "", err
	}
	if !validNamedTypeToken(packageName) {
		return "", invalid("semantic package token is invalid")
	}
	return packageName + "$" + identifier(name), nil
}
