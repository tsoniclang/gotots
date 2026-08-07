package reflectiontype

import (
	"go/types"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type reflectedMethodSet struct {
	identities []string
	members    map[string]struct{}
}

func methodSetMetadata(
	context api.Context,
	names api.ReflectionNames,
	sourceType types.Type,
) ([]tsgo.ObjectLiteralElementLike, []api.RootRequest, error) {
	source, err := reflectedMethodTokens(names, sourceType)
	if err != nil {
		return nil, nil, err
	}
	pointer, err := reflectedMethodTokens(
		names,
		types.NewPointer(sourceType),
	)
	if err != nil {
		return nil, nil, err
	}
	factory := context.Factory()
	properties := make([]tsgo.ObjectLiteralElementLike, 0, 3)
	if len(source.identities) != 0 {
		properties = append(properties, stringProperty(
			factory,
			"methodSet",
			strings.Join(source.identities, ""),
		))
	}
	pointerInherits := true
	for identity := range source.members {
		if _, exists := pointer.members[identity]; !exists {
			pointerInherits = false
			break
		}
	}
	pointerIdentities := pointer.identities
	if pointerInherits {
		pointerIdentities = make([]string, 0, len(pointer.identities))
		for _, identity := range pointer.identities {
			if _, inherited := source.members[identity]; !inherited {
				pointerIdentities = append(pointerIdentities, identity)
			}
		}
		if len(pointer.members) != 0 {
			properties = append(properties, booleanProperty(
				factory,
				"pointerInheritsMethods",
				true,
			))
		}
	}
	if len(pointerIdentities) != 0 {
		properties = append(properties, stringProperty(
			factory,
			"pointerMethodSet",
			strings.Join(pointerIdentities, ""),
		))
	}
	return properties, nil, nil
}

func reflectedMethodTokens(
	names api.ReflectionNames,
	sourceType types.Type,
) (reflectedMethodSet, error) {
	result := reflectedMethodSet{members: make(map[string]struct{})}
	methodSet := types.NewMethodSet(sourceType)
	result.identities = make([]string, 0, methodSet.Len())
	for index := range methodSet.Len() {
		method, ok := methodSet.At(index).Obj().(*types.Func)
		if !ok {
			return reflectedMethodSet{}, &api.GeneratedArtifactShapeError{
				Artifact: sourceType.String(),
				Reason:   "reflection method set contains a non-method object",
			}
		}
		identity, err := names.ReflectionMethodIdentity(method)
		if err != nil {
			return reflectedMethodSet{}, err
		}
		if _, duplicate := result.members[identity]; duplicate {
			continue
		}
		result.members[identity] = struct{}{}
		result.identities = append(result.identities, identity)
	}
	sort.Strings(result.identities)
	return result, nil
}
