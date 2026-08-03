package reflectiontype

import (
	"fmt"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	context api.Context,
	artifact *api.GeneratedArtifact,
) ([]tsgo.Statement, []api.RootRequest, error) {
	sourceType, reflectionType, ok := artifact.ReflectionType()
	if !ok {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Artifact: artifact.TargetName(),
			Reason:   "reflection-type source is invalid",
		}
	}
	names, ok := context.Names().(api.ReflectionNames)
	if !ok {
		return nil, nil, &api.ContextError{
			Reason: "reflection names are unavailable",
		}
	}
	operations, err := names.ReflectionOperations(reflectionType)
	if err != nil {
		return nil, nil, err
	}
	contract, err := context.Names().InterfaceContract(reflectionType.Type())
	if err != nil {
		return nil, nil, err
	}
	targetType, err := context.Names().TypeReference(reflectionType)
	if err != nil {
		return nil, nil, err
	}
	metadata, metadataRequests, err := metadataExpression(
		context,
		names,
		artifact.ArtifactKey(),
		sourceType,
		reflectionType,
		targetType,
	)
	if err != nil {
		return nil, nil, err
	}
	initializer := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			operations.Expression(context.Factory()),
			nil,
			context.Factory().Identifier("$create"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{
			metadata,
			context.Factory().Identifier(contract.ContractName()),
		},
		tsgo.NodeFlagsNone,
	)
	statement := context.Factory().VariableStatement(
		[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(artifact.TargetName()),
					nil,
					context.Factory().TypeReferenceNode(
						targetType.EntityName(context.Factory()),
						nil,
					),
					initializer,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
	return []tsgo.Statement{statement}, api.CombineRequests(
		operations.Requests(),
		contract.Requests(),
		targetType.Requests(),
		metadataRequests,
	), nil
}

func metadataExpression(
	context api.Context,
	names api.ReflectionNames,
	identity string,
	sourceType types.Type,
	reflectionType *types.TypeName,
	targetType api.NameReference,
) (tsgo.ObjectLiteralExpression, []api.RootRequest, error) {
	factory := context.Factory()
	description, err := describe(context.TypesSizes(), sourceType)
	if err != nil {
		return nil, nil, err
	}
	properties := []tsgo.ObjectLiteralElementLike{
		stringProperty(factory, "identity", identity),
		numberProperty(factory, "kind", int64(description.kind)),
		stringProperty(factory, "name", description.name),
		stringProperty(factory, "pkgPath", description.pkgPath),
		stringProperty(factory, "text", description.text),
		numberProperty(factory, "size", description.size),
		numberProperty(factory, "align", description.align),
		numberProperty(factory, "fieldAlign", description.fieldAlign),
		numberProperty(factory, "bits", description.bits),
		booleanProperty(factory, "comparable", types.Comparable(sourceType)),
		numberProperty(factory, "length", description.length),
		numberProperty(factory, "chanDir", description.chanDir),
		booleanProperty(factory, "variadic", description.variadic),
		stringArrayProperty(factory, "assignableTo", []string{identity}),
		stringArrayProperty(factory, "convertibleTo", []string{identity}),
		stringArrayProperty(factory, "implements", []string{identity}),
	}
	var requests []api.RootRequest
	if interfaceDynamicType(sourceType) {
		dynamicType, dynamicErr := context.Names().InterfaceDynamicType(sourceType)
		if dynamicErr != nil {
			return nil, nil, dynamicErr
		}
		properties = append(properties, expressionProperty(
			factory,
			"dynamicType",
			dynamicType.Expression(factory),
		))
		requests = append(requests, dynamicType.Requests()...)
	}
	for _, relation := range []struct {
		name      string
		typeValue types.Type
	}{
		{"elem", description.elem},
		{"key", description.key},
	} {
		if relation.typeValue == nil {
			continue
		}
		reference, relationErr := names.ReflectionType(
			relation.typeValue,
			reflectionType,
		)
		if relationErr != nil {
			return nil, nil, relationErr
		}
		properties = append(properties, expressionProperty(
			factory,
			relation.name,
			arrow(factory, targetType, reference.Expression(factory)),
		))
		requests = append(requests, reference.Requests()...)
	}
	fields, fieldRequests, err := fieldMetadata(
		factory,
		names,
		reflectionType,
		targetType,
		description.structType,
		description.offsets,
	)
	if err != nil {
		return nil, nil, err
	}
	if fields != nil {
		properties = append(properties, expressionProperty(
			factory,
			"fields",
			fields,
		))
		requests = append(requests, fieldRequests...)
	}
	return factory.ObjectLiteralExpression(properties, true), requests, nil
}

type typeDescription struct {
	kind       int
	name       string
	pkgPath    string
	text       string
	size       int64
	align      int64
	fieldAlign int64
	bits       int64
	length     int64
	chanDir    int64
	variadic   bool
	elem       types.Type
	key        types.Type
	structType *types.Struct
	offsets    []int64
}

func describe(sizes types.Sizes, source types.Type) (typeDescription, error) {
	if sizes == nil || source == nil {
		return typeDescription{}, fmt.Errorf("describe reflection type: source is invalid")
	}
	result := typeDescription{
		text:       types.TypeString(source, packageQualifier),
		size:       sizes.Sizeof(source),
		align:      int64(sizes.Alignof(source)),
		fieldAlign: int64(sizes.Alignof(source)),
	}
	underlying := types.Unalias(source).Underlying()
	if named, ok := types.Unalias(source).(*types.Named); ok && named.Obj() != nil {
		result.name = named.Obj().Name()
		if named.Obj().Pkg() != nil {
			result.pkgPath = named.Obj().Pkg().Path()
		}
	}
	switch selected := underlying.(type) {
	case *types.Basic:
		result.kind, result.bits = basicKind(sizes, selected)
	case *types.Array:
		result.kind, result.length, result.elem = 17, selected.Len(), selected.Elem()
	case *types.Chan:
		result.kind, result.elem = 18, selected.Elem()
		switch selected.Dir() {
		case types.RecvOnly:
			result.chanDir = 1
		case types.SendOnly:
			result.chanDir = 2
		default:
			result.chanDir = 3
		}
	case *types.Signature:
		result.kind, result.variadic = 19, selected.Variadic()
	case *types.Interface:
		result.kind = 20
	case *types.Map:
		result.kind, result.key, result.elem = 21, selected.Key(), selected.Elem()
	case *types.Pointer:
		result.kind, result.elem = 22, selected.Elem()
	case *types.Slice:
		result.kind, result.elem = 23, selected.Elem()
	case *types.Struct:
		result.kind, result.structType = 25, selected
		fields := make([]*types.Var, selected.NumFields())
		for index := range selected.NumFields() {
			fields[index] = selected.Field(index)
		}
		result.offsets = sizes.Offsetsof(fields)
	default:
		return typeDescription{}, fmt.Errorf(
			"describe reflection type %q: unsupported Go type %T",
			result.text,
			underlying,
		)
	}
	return result, nil
}

func basicKind(sizes types.Sizes, source *types.Basic) (int, int64) {
	bits := sizes.Sizeof(source) * 8
	switch source.Kind() {
	case types.Bool:
		return 1, 0
	case types.Int:
		return 2, bits
	case types.Int8:
		return 3, 8
	case types.Int16:
		return 4, 16
	case types.Int32:
		return 5, 32
	case types.Int64:
		return 6, 64
	case types.Uint:
		return 7, bits
	case types.Uint8:
		return 8, 8
	case types.Uint16:
		return 9, 16
	case types.Uint32:
		return 10, 32
	case types.Uint64:
		return 11, 64
	case types.Uintptr:
		return 12, bits
	case types.Float32:
		return 13, 32
	case types.Float64:
		return 14, 64
	case types.Complex64:
		return 15, 64
	case types.Complex128:
		return 16, 128
	case types.String:
		return 24, 0
	case types.UnsafePointer:
		return 26, 0
	default:
		return 0, 0
	}
}

func fieldMetadata(
	factory tsgo.Factory,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	targetType api.NameReference,
	structType *types.Struct,
	offsets []int64,
) (tsgo.ArrayLiteralExpression, []api.RootRequest, error) {
	if structType == nil {
		return nil, nil, nil
	}
	fields := make([]tsgo.Expression, 0, structType.NumFields())
	var requests []api.RootRequest
	for index := range structType.NumFields() {
		field := structType.Field(index)
		reference, err := names.ReflectionType(field.Type(), reflectionType)
		if err != nil {
			return nil, nil, err
		}
		pkgPath := ""
		if !field.Exported() && field.Pkg() != nil {
			pkgPath = field.Pkg().Path()
		}
		fields = append(fields, factory.ObjectLiteralExpression(
			[]tsgo.ObjectLiteralElementLike{
				stringProperty(factory, "name", field.Name()),
				stringProperty(factory, "pkgPath", pkgPath),
				expressionProperty(
					factory,
					"type",
					arrow(factory, targetType, reference.Expression(factory)),
				),
				stringProperty(factory, "tag", structType.Tag(index)),
				numberProperty(factory, "offset", offsets[index]),
				expressionProperty(factory, "index", factory.ArrayLiteralExpression(
					[]tsgo.Expression{factory.NumericLiteral(strconv.Itoa(index), tsgo.TokenFlagsNone)},
					false,
				)),
				booleanProperty(factory, "anonymous", field.Embedded()),
			},
			true,
		))
		requests = append(requests, reference.Requests()...)
	}
	return factory.ArrayLiteralExpression(fields, true), requests, nil
}

func packageQualifier(source *types.Package) string {
	if source == nil {
		return ""
	}
	return source.Name()
}

func interfaceDynamicType(sourceType types.Type) bool {
	switch types.Unalias(sourceType).Underlying().(type) {
	case *types.Interface, *types.Tuple, *types.TypeParam, *types.Union:
		return false
	default:
		return true
	}
}

func stringProperty(factory tsgo.Factory, name string, value string) tsgo.PropertyAssignment {
	return typedExpressionProperty(
		factory,
		name,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
		factory.StringLiteral(value, tsgo.TokenFlagsNone),
	)
}

func numberProperty(factory tsgo.Factory, name string, value int64) tsgo.PropertyAssignment {
	return typedExpressionProperty(
		factory,
		name,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.NumericLiteral(strconv.FormatInt(value, 10), tsgo.TokenFlagsNone),
	)
}

func booleanProperty(factory tsgo.Factory, name string, value bool) tsgo.PropertyAssignment {
	expression := tsgo.Expression(factory.FalseLiteral())
	if value {
		expression = factory.TrueLiteral()
	}
	return typedExpressionProperty(
		factory,
		name,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		expression,
	)
}

func stringArrayProperty(
	factory tsgo.Factory,
	name string,
	values []string,
) tsgo.PropertyAssignment {
	expressions := make([]tsgo.Expression, 0, len(values))
	for _, value := range values {
		expressions = append(expressions, factory.StringLiteral(value, tsgo.TokenFlagsNone))
	}
	return typedExpressionProperty(
		factory,
		name,
		factory.ArrayTypeNode(
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
		),
		factory.ArrayLiteralExpression(expressions, false),
	)
}

func expressionProperty(
	factory tsgo.Factory,
	name string,
	value tsgo.Expression,
) tsgo.PropertyAssignment {
	return typedExpressionProperty(
		factory,
		name,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindObjectKeyword),
		value,
	)
}

func typedExpressionProperty(
	factory tsgo.Factory,
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.PropertyAssignment {
	return factory.PropertyAssignment(
		nil,
		factory.Identifier(name),
		nil,
		targetType,
		value,
	)
}

func arrow(
	factory tsgo.Factory,
	returnType api.NameReference,
	value tsgo.Expression,
) tsgo.ArrowFunction {
	return factory.ArrowFunction(
		nil,
		nil,
		nil,
		factory.TypeReferenceNode(returnType.EntityName(factory), nil),
		factory.EqualsGreaterThanToken(),
		factory.Block(
			[]tsgo.Statement{factory.ReturnStatement(value)},
			true,
		),
	)
}
