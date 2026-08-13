package semanticname

import (
	"go/types"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

func ConcretizationSuffix(
	arguments []types.Type,
	synchronous bool,
) (string, error) {
	return concretizationSuffix(
		arguments,
		synchronous,
		typeIdentityResolver{},
	)
}

func OperationName(
	operation string,
	method *types.Func,
	signature *types.Signature,
) (string, error) {
	return operationName(
		operation,
		method,
		signature,
		typeIdentityResolver{},
	)
}

func CapabilityModule(operation string) (string, error) {
	if operation == "" {
		return "", invalid("generic capability operation is empty")
	}
	return operationIdentifier(operation)
}

func ConcretizationModule(owner *types.Func) (string, error) {
	if owner == nil || owner.Origin() != owner || owner.Pkg() == nil {
		return "", invalid("generic concretization owner is invalid")
	}
	signature, ok := owner.Type().(*types.Signature)
	if !ok {
		return "", invalid("generic concretization owner has no signature")
	}
	segments := strings.Split(owner.Pkg().Path(), "/")
	for index, segment := range segments {
		if segment == "" {
			return "", invalid("generic concretization package path is invalid")
		}
		segments[index] = identifier(segment)
	}
	name := identifier(owner.Name())
	if signature.Recv() != nil {
		receiver, err := receiverName(signature.Recv().Type())
		if err != nil {
			return "", err
		}
		name = receiver + "$" + name
	}
	return strings.Join(append(segments, name), "/"), nil
}

func ConcretizationName(owner *types.Func, suffix string) (string, error) {
	if owner == nil || owner.Origin() != owner || suffix == "" {
		return "", invalid("generic concretization name owner is invalid")
	}
	signature, ok := owner.Type().(*types.Signature)
	if !ok {
		return "", invalid("generic concretization name owner has no signature")
	}
	name := identifier(owner.Name())
	if signature.Recv() != nil {
		receiver, err := receiverName(signature.Recv().Type())
		if err != nil {
			return "", err
		}
		name = receiver + "$" + name
	}
	return name + suffix, nil
}

func Type(source types.Type) (string, error) {
	return typeWithIdentity(source, typeIdentityResolver{})
}

func TypeWithLocalIdentity(
	source types.Type,
	localIdentity LocalNamedIdentity,
) (string, error) {
	if localIdentity == nil {
		return "", invalid("semantic local identity owner is nil")
	}
	return typeWithIdentity(source, typeIdentityResolver{
		localIdentity: localIdentity,
	})
}

func TypeWithIdentityTokens(
	source types.Type,
	namedToken NamedTypeToken,
	packageToken PackageToken,
) (string, error) {
	if namedToken == nil || packageToken == nil {
		return "", invalid("semantic identity-token owner is nil")
	}
	return typeWithIdentity(source, typeIdentityResolver{
		namedToken:   namedToken,
		packageToken: packageToken,
	})
}

func Identifier(source string) string {
	return identifier(source)
}

func typeWithIdentity(
	source types.Type,
	identity typeIdentityResolver,
) (string, error) {
	if source == nil {
		return "", invalid("semantic type is nil")
	}
	return typeToken(
		types.Unalias(source),
		make(map[types.Type]struct{}),
		make(map[*types.TypeParam]int),
		identity,
	)
}

func typeToken(
	source types.Type,
	pending map[types.Type]struct{},
	parameters map[*types.TypeParam]int,
	identity typeIdentityResolver,
) (string, error) {
	source = types.Unalias(source)
	switch selected := source.(type) {
	case *types.Basic:
		return identifier(selected.Name()), nil
	case *types.Named:
		name, err := namedTypeName(selected.Obj(), identity)
		if err != nil {
			return "", err
		}
		arguments, err := typeListTokens(
			selected.TypeArgs(),
			pending,
			parameters,
			identity,
		)
		if err != nil || len(arguments) == 0 {
			return "Named_" + name, err
		}
		return "Named_" + name + "Of_" + strings.Join(arguments, "_And_"), nil
	case *types.Pointer:
		return unaryTypeToken(
			"PointerTo_",
			selected.Elem(),
			pending,
			parameters,
			identity,
		)
	case *types.Slice:
		return unaryTypeToken(
			"SliceOf_",
			selected.Elem(),
			pending,
			parameters,
			identity,
		)
	case *types.Array:
		return unaryTypeToken(
			"Array"+strconv.FormatInt(selected.Len(), 10)+"Of_",
			selected.Elem(),
			pending,
			parameters,
			identity,
		)
	case *types.Map:
		key, err := typeToken(
			selected.Key(),
			pending,
			parameters,
			identity,
		)
		if err != nil {
			return "", err
		}
		element, err := typeToken(
			selected.Elem(),
			pending,
			parameters,
			identity,
		)
		if err != nil {
			return "", err
		}
		return "MapOf_" + key + "_To_" + element, nil
	case *types.Chan:
		prefix := "ChannelOf_"
		switch selected.Dir() {
		case types.SendOnly:
			prefix = "SendChannelOf_"
		case types.RecvOnly:
			prefix = "ReceiveChannelOf_"
		}
		return unaryTypeToken(
			prefix,
			selected.Elem(),
			pending,
			parameters,
			identity,
		)
	case *types.Signature:
		return signatureTokenWithPending(
			selected,
			pending,
			parameters,
			identity,
		)
	case *types.Struct:
		return structToken(selected, pending, parameters, identity)
	case *types.Interface:
		return interfaceToken(selected, pending, parameters, identity)
	case *types.Tuple:
		values, err := tupleTokens(
			selected,
			pending,
			parameters,
			identity,
		)
		if err != nil {
			return "", err
		}
		return "Tuple_" + joinedOrVoid(values), nil
	case *types.TypeParam:
		object := selected.Obj()
		if object == nil {
			return "", invalid("semantic type parameter has no object")
		}
		index := selected.Index()
		if index < 0 {
			var ok bool
			index, ok = parameters[selected]
			if ok {
				return "T" + strconv.Itoa(index), nil
			}
			index = len(parameters)
			parameters[selected] = index
		}
		return "T" + strconv.Itoa(index), nil
	case *types.Union:
		terms := make([]string, 0, selected.Len())
		for index := range selected.Len() {
			term := selected.Term(index)
			value, err := typeToken(
				term.Type(),
				pending,
				parameters,
				identity,
			)
			if err != nil {
				return "", err
			}
			if term.Tilde() {
				value = "Underlying_" + value
			}
			terms = append(terms, value)
		}
		slices.Sort(terms)
		return "Union_" + joinedOrVoid(terms), nil
	default:
		return "", invalid("unsupported semantic type " + types.TypeString(source, nil))
	}
}

func signatureToken(signature *types.Signature) (string, error) {
	return signatureTokenWithPending(
		signature,
		make(map[types.Type]struct{}),
		make(map[*types.TypeParam]int),
		typeIdentityResolver{},
	)
}

func signatureTokenWithPending(
	signature *types.Signature,
	pending map[types.Type]struct{},
	parameters map[*types.TypeParam]int,
	identity typeIdentityResolver,
) (string, error) {
	parameterTokens, err := tupleTokens(
		signature.Params(),
		pending,
		parameters,
		identity,
	)
	if err != nil {
		return "", err
	}
	if signature.Variadic() && len(parameterTokens) != 0 {
		parameterTokens[len(parameterTokens)-1] = "Variadic_" + parameterTokens[len(parameterTokens)-1]
	}
	results, err := tupleTokens(
		signature.Results(),
		pending,
		parameters,
		identity,
	)
	if err != nil {
		return "", err
	}
	return joinedOrVoid(parameterTokens) + "_to_" + joinedOrVoid(results), nil
}

func structToken(
	structure *types.Struct,
	pending map[types.Type]struct{},
	parameters map[*types.TypeParam]int,
	identity typeIdentityResolver,
) (string, error) {
	fields := make([]string, 0, structure.NumFields())
	for index := range structure.NumFields() {
		field := structure.Field(index)
		fieldType, err := typeToken(
			field.Type(),
			pending,
			parameters,
			identity,
		)
		if err != nil {
			return "", err
		}
		fieldName := field.Name()
		if !field.Exported() && field.Pkg() != nil {
			fieldName, err = packageObjectName(
				field.Pkg(),
				fieldName,
				identity,
			)
			if err != nil {
				return "", err
			}
		}
		kind := "Field_"
		if field.Embedded() {
			kind = "Embedded_"
		}
		fields = append(fields, kind+identifier(fieldName)+"_"+fieldType+
			"_Tag_"+identifier(structure.Tag(index)))
	}
	return "Struct_" + joinedOrVoid(fields), nil
}

func interfaceToken(
	contract *types.Interface,
	pending map[types.Type]struct{},
	parameters map[*types.TypeParam]int,
	identity typeIdentityResolver,
) (string, error) {
	if _, recursive := pending[contract]; recursive {
		return "", invalid("recursive anonymous interface has no exact semantic name")
	}
	pending[contract] = struct{}{}
	defer delete(pending, contract)
	contract = contract.Complete()
	if !contract.IsMethodSet() {
		return "", invalid("constraint interface has no runtime semantic name")
	}
	parts := make([]string, 0, contract.NumMethods())
	for index := range contract.NumMethods() {
		method := contract.Method(index)
		signature, ok := method.Type().(*types.Signature)
		if !ok {
			return "", invalid("interface method has no signature")
		}
		methodType, err := signatureTokenWithPending(
			signature,
			pending,
			parameters,
			identity,
		)
		if err != nil {
			return "", err
		}
		methodName, err := packageObjectName(
			method.Pkg(),
			method.Name(),
			identity,
		)
		if err != nil {
			return "", err
		}
		parts = append(parts, "Method_"+methodName+"_"+methodType)
	}
	slices.Sort(parts)
	return "Interface_" + joinedOrVoid(parts), nil
}

func typeListTokens(
	list *types.TypeList,
	pending map[types.Type]struct{},
	parameters map[*types.TypeParam]int,
	identity typeIdentityResolver,
) ([]string, error) {
	if list == nil {
		return nil, nil
	}
	values := make([]string, 0, list.Len())
	for index := range list.Len() {
		value, err := typeToken(
			list.At(index),
			pending,
			parameters,
			identity,
		)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func tupleTokens(
	tuple *types.Tuple,
	pending map[types.Type]struct{},
	parameters map[*types.TypeParam]int,
	identity typeIdentityResolver,
) ([]string, error) {
	if tuple == nil {
		return nil, nil
	}
	values := make([]string, 0, tuple.Len())
	for index := range tuple.Len() {
		value, err := typeToken(
			tuple.At(index).Type(),
			pending,
			parameters,
			identity,
		)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func unaryTypeToken(
	prefix string,
	element types.Type,
	pending map[types.Type]struct{},
	parameters map[*types.TypeParam]int,
	identity typeIdentityResolver,
) (string, error) {
	value, err := typeToken(
		element,
		pending,
		parameters,
		identity,
	)
	return prefix + value, err
}

func namedTypeName(
	object *types.TypeName,
	identity typeIdentityResolver,
) (string, error) {
	if object == nil {
		return "", invalid("semantic named type has no object")
	}
	if object.Pkg() == nil {
		return identifier(object.Name()), nil
	}
	if identity.namedToken != nil {
		token, err := identity.namedToken(object)
		if err != nil {
			return "", err
		}
		if !validNamedTypeToken(token) {
			return "", invalid("semantic named-type token is invalid")
		}
		return token, nil
	}
	if object.Parent() != nil && object.Parent() != object.Pkg().Scope() {
		if identity.localIdentity == nil {
			return identifier(object.Pkg().Path()) + "_" +
				identifier(object.Name()), nil
		}
		localIdentity, err := identity.localIdentity(object)
		if err != nil {
			return "", err
		}
		if localIdentity == "" {
			return "", invalid("local semantic named type identity is empty")
		}
		return identifier(localIdentity), nil
	}
	return identifier(object.Pkg().Path()) + "_" + identifier(object.Name()), nil
}

func validNamedTypeToken(token string) bool {
	if token == "" {
		return false
	}
	for _, value := range token {
		if value >= 'A' && value <= 'Z' ||
			value >= 'a' && value <= 'z' ||
			value >= '0' && value <= '9' ||
			value == '_' || value == '$' {
			continue
		}
		return false
	}
	return true
}

func receiverName(source types.Type) (string, error) {
	if pointer, ok := types.Unalias(source).(*types.Pointer); ok {
		source = pointer.Elem()
	}
	named, ok := types.Unalias(source).(*types.Named)
	if !ok || named.Obj() == nil {
		return "", invalid("generic method receiver is not named")
	}
	return identifier(named.Obj().Name()), nil
}

func joinedOrVoid(values []string) string {
	if len(values) == 0 {
		return "void"
	}
	return strings.Join(values, "_")
}

func identifier(source string) string {
	if source == "" {
		return "_empty_"
	}
	var target strings.Builder
	for len(source) != 0 {
		value, width := utf8.DecodeRuneInString(source)
		source = source[width:]
		if value >= 'A' && value <= 'Z' ||
			value >= 'a' && value <= 'z' ||
			value >= '0' && value <= '9' {
			target.WriteRune(value)
			continue
		}
		target.WriteString("_u")
		target.WriteString(strconv.FormatInt(int64(value), 16))
		target.WriteByte('_')
	}
	return target.String()
}

func operationIdentifier(source string) (string, error) {
	for _, value := range source {
		if value >= 'a' && value <= 'z' ||
			value >= '0' && value <= '9' ||
			value == '_' {
			continue
		}
		return "", invalid("generic operation name is not canonical")
	}
	if source == "" {
		return "", invalid("generic operation name is empty")
	}
	return source, nil
}
