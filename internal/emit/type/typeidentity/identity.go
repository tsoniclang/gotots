package typeidentity

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type NamedObjectIdentity func(*types.TypeName) (string, error)

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
	if sourceType == nil {
		return "", &api.NameError{
			Reason: "generated-artifact Go type is nil",
		}
	}
	if namedObjectIdentity == nil {
		return "", &api.NameError{
			Reason: "generated-artifact named-object identity owner is nil",
		}
	}
	var descriptor strings.Builder
	if err := appendTypeDescriptor(
		&descriptor,
		sourceType,
		namedObjectIdentity,
	); err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(descriptor.String()))
	return hex.EncodeToString(digest[:]), nil
}

func appendTypeDescriptor(
	target *strings.Builder,
	sourceType types.Type,
	namedObjectIdentity NamedObjectIdentity,
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
		identity, err := namedObjectIdentity(object)
		if err != nil {
			return err
		}
		appendPart(target, identity)
		appendPart(target, strconv.Itoa(source.TypeArgs().Len()))
		for index := range source.TypeArgs().Len() {
			if err := appendTypeDescriptor(
				target,
				source.TypeArgs().At(index),
				namedObjectIdentity,
			); err != nil {
				return err
			}
		}
	case *types.Pointer:
		appendPart(target, "pointer")
		return appendTypeDescriptor(target, source.Elem(), namedObjectIdentity)
	case *types.Slice:
		appendPart(target, "slice")
		return appendTypeDescriptor(target, source.Elem(), namedObjectIdentity)
	case *types.Array:
		appendPart(target, "array")
		appendPart(target, strconv.FormatInt(source.Len(), 10))
		return appendTypeDescriptor(target, source.Elem(), namedObjectIdentity)
	case *types.Map:
		appendPart(target, "map")
		if err := appendTypeDescriptor(
			target,
			source.Key(),
			namedObjectIdentity,
		); err != nil {
			return err
		}
		return appendTypeDescriptor(target, source.Elem(), namedObjectIdentity)
	case *types.Chan:
		appendPart(target, "channel")
		appendPart(target, strconv.Itoa(int(source.Dir())))
		return appendTypeDescriptor(target, source.Elem(), namedObjectIdentity)
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
				namedObjectIdentity,
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
			namedObjectIdentity,
		); err != nil {
			return err
		}
		return appendTupleDescriptor(
			target,
			source.Results(),
			namedObjectIdentity,
		)
	default:
		return &api.NameError{
			Reason: "Go type has no deterministic generated-artifact key",
		}
	}
	return nil
}

func appendTupleDescriptor(
	target *strings.Builder,
	tuple *types.Tuple,
	namedObjectIdentity NamedObjectIdentity,
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
			namedObjectIdentity,
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
