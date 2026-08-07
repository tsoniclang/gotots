package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type reflectedMethodSet struct {
	names      map[string]struct{}
	tokenNames []string
	tokens     []tsgo.Expression
	requests   []api.RootRequest
}

func methodSetMetadata(
	context api.Context,
	sourceType types.Type,
) ([]tsgo.ObjectLiteralElementLike, []api.RootRequest, error) {
	source, err := reflectedMethodTokens(context, sourceType)
	if err != nil {
		return nil, nil, err
	}
	pointer, err := reflectedMethodTokens(
		context,
		types.NewPointer(sourceType),
	)
	if err != nil {
		return nil, nil, err
	}
	factory := context.Factory()
	properties := make([]tsgo.ObjectLiteralElementLike, 0, 3)
	if len(source.tokens) != 0 {
		properties = append(properties, expressionProperty(
			factory,
			"methodTokens",
			factory.ArrayLiteralExpression(source.tokens, false),
		))
	}
	pointerInherits := true
	for name := range source.names {
		if _, exists := pointer.names[name]; !exists {
			pointerInherits = false
			break
		}
	}
	pointerTokens := pointer.tokens
	if pointerInherits {
		pointerTokens = make([]tsgo.Expression, 0, len(pointer.tokens))
		for index, token := range pointer.tokens {
			if _, inherited := source.names[pointer.tokenNames[index]]; !inherited {
				pointerTokens = append(pointerTokens, token)
			}
		}
		if len(pointer.names) != 0 {
			properties = append(properties, booleanProperty(
				factory,
				"pointerInheritsMethods",
				true,
			))
		}
	}
	if len(pointerTokens) != 0 {
		properties = append(properties, expressionProperty(
			factory,
			"pointerMethodTokens",
			factory.ArrayLiteralExpression(pointerTokens, false),
		))
	}
	return properties, api.CombineRequests(
		source.requests,
		pointer.requests,
	), nil
}

func reflectedMethodTokens(
	context api.Context,
	sourceType types.Type,
) (reflectedMethodSet, error) {
	result := reflectedMethodSet{names: make(map[string]struct{})}
	methodSet := types.NewMethodSet(sourceType)
	result.tokens = make([]tsgo.Expression, 0, methodSet.Len())
	result.tokenNames = make([]string, 0, methodSet.Len())
	for index := range methodSet.Len() {
		method, ok := methodSet.At(index).Obj().(*types.Func)
		if !ok {
			return reflectedMethodSet{}, &api.GeneratedArtifactShapeError{
				Artifact: sourceType.String(),
				Reason:   "reflection method set contains a non-method object",
			}
		}
		reference, err := context.Names().InterfaceMethodToken(method)
		if err != nil {
			return reflectedMethodSet{}, err
		}
		if _, duplicate := result.names[reference.Name()]; duplicate {
			continue
		}
		result.names[reference.Name()] = struct{}{}
		result.tokenNames = append(result.tokenNames, reference.Name())
		result.tokens = append(
			result.tokens,
			reference.Expression(context.Factory()),
		)
		result.requests = append(result.requests, reference.Requests()...)
	}
	return result, nil
}
