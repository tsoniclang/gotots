package typeidentity

import (
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/types"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type NamedObjectIdentity func(*types.TypeName) (string, error)

type TypeParameterIdentity func(*types.TypeParam) (string, error)

type identityOwner struct {
	namedObject   NamedObjectIdentity
	typeParameter TypeParameterIdentity
}

func NamedObjectKey(
	object *types.TypeName,
	sourceFile *ast.File,
	sourcePath string,
) (string, error) {
	if object == nil || object.Pkg() == nil {
		return "", &api.NameError{
			Reason: "generated-artifact named component has no package identity",
		}
	}
	if object.Parent() == object.Pkg().Scope() {
		return object.Pkg().Path() + "|" + object.Name(), nil
	}
	if sourceFile == nil ||
		sourcePath == "" ||
		object.Pos() < sourceFile.Pos() ||
		object.Pos() > sourceFile.End() {
		return "", &api.NameError{
			Name:   object.Name(),
			Reason: "generated-artifact local component has no lexical identity",
		}
	}
	offset := int64(object.Pos() - sourceFile.Pos())
	return object.Pkg().Path() + "|" +
		sourcePath + "|" +
		object.Name() + "|" +
		strconv.FormatInt(offset, 10), nil
}

func LocalComponent(sourceType types.Type) (*types.TypeName, bool) {
	components := LocalComponents(sourceType)
	if len(components) == 0 {
		return nil, false
	}
	return components[0], true
}

func LocalComponents(sourceType types.Type) []*types.TypeName {
	seen := make(map[*types.TypeName]struct{})
	var components []*types.TypeName
	collectLocalComponents(sourceType, seen, &components)
	return components
}

func collectLocalComponents(
	sourceType types.Type,
	seen map[*types.TypeName]struct{},
	components *[]*types.TypeName,
) {
	sourceType = types.Unalias(sourceType)
	switch source := sourceType.(type) {
	case *types.Named:
		object := source.Obj()
		if object != nil &&
			object.Pkg() != nil &&
			object.Parent() != object.Pkg().Scope() {
			if _, duplicate := seen[object]; !duplicate {
				seen[object] = struct{}{}
				*components = append(*components, object)
			}
		}
		for index := range source.TypeArgs().Len() {
			collectLocalComponents(
				source.TypeArgs().At(index),
				seen,
				components,
			)
		}
	case *types.Pointer:
		collectLocalComponents(source.Elem(), seen, components)
	case *types.Slice:
		collectLocalComponents(source.Elem(), seen, components)
	case *types.Array:
		collectLocalComponents(source.Elem(), seen, components)
	case *types.Map:
		collectLocalComponents(source.Key(), seen, components)
		collectLocalComponents(source.Elem(), seen, components)
	case *types.Chan:
		collectLocalComponents(source.Elem(), seen, components)
	case *types.Struct:
		for index := range source.NumFields() {
			collectLocalComponents(
				source.Field(index).Type(),
				seen,
				components,
			)
		}
	case *types.Signature:
		collectLocalTupleComponents(source.Params(), seen, components)
		collectLocalTupleComponents(source.Results(), seen, components)
	case *types.Interface:
		source = source.Complete()
		for index := range source.NumMethods() {
			collectLocalComponents(
				source.Method(index).Type(),
				seen,
				components,
			)
		}
	}
}

func collectLocalTupleComponents(
	tuple *types.Tuple,
	seen map[*types.TypeName]struct{},
	components *[]*types.TypeName,
) {
	if tuple == nil {
		return
	}
	for index := range tuple.Len() {
		collectLocalComponents(
			tuple.At(index).Type(),
			seen,
			components,
		)
	}
}

func BuildKey(
	sourceType types.Type,
	namedObjectIdentity NamedObjectIdentity,
) (string, error) {
	return buildKey(sourceType, identityOwner{
		namedObject: namedObjectIdentity,
	})
}

func BuildParameterizedKey(
	sourceType types.Type,
	namedObjectIdentity NamedObjectIdentity,
	typeParameterIdentity TypeParameterIdentity,
) (string, error) {
	if typeParameterIdentity == nil {
		return "", &api.NameError{
			Reason: "generated-artifact type-parameter identity owner is nil",
		}
	}
	return buildKey(sourceType, identityOwner{
		namedObject:   namedObjectIdentity,
		typeParameter: typeParameterIdentity,
	})
}

func buildKey(
	sourceType types.Type,
	owner identityOwner,
) (string, error) {
	if sourceType == nil {
		return "", &api.NameError{
			Reason: "generated-artifact Go type is nil",
		}
	}
	if owner.namedObject == nil {
		return "", &api.NameError{
			Reason: "generated-artifact named-object identity owner is nil",
		}
	}
	var descriptor strings.Builder
	if err := appendTypeDescriptor(
		&descriptor,
		sourceType,
		owner,
	); err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(descriptor.String()))
	return hex.EncodeToString(digest[:]), nil
}

func appendTypeDescriptor(
	target *strings.Builder,
	sourceType types.Type,
	owner identityOwner,
) error {
	sourceType = types.Unalias(sourceType)
	switch source := sourceType.(type) {
	case *types.Basic:
		appendPart(target, "basic")
		appendPart(target, strconv.Itoa(int(source.Kind())))
	case *types.Named:
		appendPart(target, "named")
		object := source.Obj()
		if object == nil {
			return &api.NameError{
				Reason: "generated-artifact named type has no declaration",
			}
		}
		identity, err := owner.namedObject(object)
		if err != nil {
			return err
		}
		appendPart(target, identity)
		appendPart(target, strconv.Itoa(source.TypeArgs().Len()))
		for index := range source.TypeArgs().Len() {
			if err := appendTypeDescriptor(
				target,
				source.TypeArgs().At(index),
				owner,
			); err != nil {
				return err
			}
		}
	case *types.TypeParam:
		if owner.typeParameter == nil {
			name := ""
			if source.Obj() != nil {
				name = source.Obj().Name()
			}
			return &api.NameError{
				Name:   name,
				Reason: "generated-artifact type parameter has no identity owner",
			}
		}
		identity, err := owner.typeParameter(source)
		if err != nil {
			return err
		}
		if identity == "" {
			name := ""
			if source.Obj() != nil {
				name = source.Obj().Name()
			}
			return &api.NameError{
				Name:   name,
				Reason: "generated-artifact type parameter has empty identity",
			}
		}
		appendPart(target, "type-parameter")
		appendPart(target, identity)
	case *types.Pointer:
		appendPart(target, "pointer")
		return appendTypeDescriptor(target, source.Elem(), owner)
	case *types.Slice:
		appendPart(target, "slice")
		return appendTypeDescriptor(target, source.Elem(), owner)
	case *types.Array:
		appendPart(target, "array")
		appendPart(target, strconv.FormatInt(source.Len(), 10))
		return appendTypeDescriptor(target, source.Elem(), owner)
	case *types.Map:
		appendPart(target, "map")
		if err := appendTypeDescriptor(
			target,
			source.Key(),
			owner,
		); err != nil {
			return err
		}
		return appendTypeDescriptor(target, source.Elem(), owner)
	case *types.Chan:
		appendPart(target, "channel")
		appendPart(target, strconv.Itoa(int(source.Dir())))
		return appendTypeDescriptor(target, source.Elem(), owner)
	case *types.Struct:
		appendPart(target, "struct")
		appendPart(target, strconv.Itoa(source.NumFields()))
		for index := range source.NumFields() {
			field := source.Field(index)
			appendPart(target, field.Name())
			packagePath := ""
			if !field.Exported() && field.Pkg() != nil {
				packagePath = field.Pkg().Path()
			}
			appendPart(target, packagePath)
			appendPart(target, strconv.FormatBool(field.Embedded()))
			appendPart(target, source.Tag(index))
			if err := appendTypeDescriptor(
				target,
				field.Type(),
				owner,
			); err != nil {
				return err
			}
		}
	case *types.Signature:
		appendPart(target, "signature")
		appendPart(target, strconv.FormatBool(source.Variadic()))
		if err := appendTupleDescriptor(
			target,
			source.Params(),
			owner,
		); err != nil {
			return err
		}
		return appendTupleDescriptor(
			target,
			source.Results(),
			owner,
		)
	case *types.Interface:
		source = source.Complete()
		if !source.IsMethodSet() {
			return &api.NameError{
				Reason: "constraint interface has no runtime generated-artifact identity",
			}
		}
		appendPart(target, "interface")
		appendPart(target, strconv.Itoa(source.NumMethods()))
		for index := range source.NumMethods() {
			method := source.Method(index)
			appendPart(target, method.Name())
			packagePath := ""
			if !method.Exported() && method.Pkg() != nil {
				packagePath = method.Pkg().Path()
			}
			appendPart(target, packagePath)
			signature, ok := method.Type().(*types.Signature)
			if !ok {
				return &api.NameError{
					Reason: "interface method has no signature identity",
				}
			}
			if err := appendTypeDescriptor(
				target,
				receiverFreeSignature(signature),
				owner,
			); err != nil {
				return err
			}
		}
	default:
		return &api.NameError{
			Reason: "Go type has no deterministic generated-artifact key",
		}
	}
	return nil
}

func receiverFreeSignature(
	signature *types.Signature,
) *types.Signature {
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		signature.Params(),
		signature.Results(),
		signature.Variadic(),
	)
}

func appendTupleDescriptor(
	target *strings.Builder,
	tuple *types.Tuple,
	owner identityOwner,
) error {
	if tuple == nil {
		appendPart(target, "0")
		return nil
	}
	appendPart(target, strconv.Itoa(tuple.Len()))
	for index := range tuple.Len() {
		if err := appendTypeDescriptor(
			target,
			tuple.At(index).Type(),
			owner,
		); err != nil {
			return err
		}
	}
	return nil
}

func appendPart(target *strings.Builder, value string) {
	target.WriteString(strconv.Itoa(len(value)))
	target.WriteByte(':')
	target.WriteString(value)
}
