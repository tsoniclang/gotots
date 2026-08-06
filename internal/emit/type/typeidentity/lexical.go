package typeidentity

import (
	"go/types"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func LexicalNamedObjectKey(
	object *types.TypeName,
	owner api.ArtifactOwner,
	root *types.Scope,
) (string, error) {
	if object == nil ||
		object.Pkg() == nil ||
		root == nil ||
		object.Parent() == nil ||
		object.Parent() == object.Pkg().Scope() ||
		object.Parent().Lookup(object.Name()) != object {
		return "", &api.NameError{
			Name:   objectName(object),
			Reason: "generated-artifact local component has no lexical identity",
		}
	}
	ownerKey, err := lexicalArtifactOwnerKey(owner, object.Pkg())
	if err != nil {
		return "", err
	}
	rootPath, err := lexicalScopePath(root, object.Pkg().Scope())
	if err != nil {
		return "", &api.NameError{
			Name:   object.Name(),
			Reason: "generated-artifact lexical root is foreign to its package",
		}
	}
	objectPath, err := lexicalScopePath(object.Parent(), root)
	if err != nil {
		return "", &api.NameError{
			Name:   object.Name(),
			Reason: "generated-artifact local component has a foreign lexical owner",
		}
	}
	var identity strings.Builder
	appendPart(&identity, "lexical")
	appendPart(&identity, ownerKey)
	appendScopePath(&identity, rootPath)
	appendPart(&identity, object.Name())
	appendScopePath(&identity, objectPath)
	return identity.String(), nil
}

func lexicalArtifactOwnerKey(
	owner api.ArtifactOwner,
	sourcePackage *types.Package,
) (string, error) {
	if sourcePackage == nil || !owner.Valid() ||
		owner.Package() != sourcePackage {
		return "", &api.NameError{
			Reason: "generated-artifact lexical owner has a foreign package",
		}
	}
	if source, ok := owner.Source(); ok {
		return sourceArtifactOwnerKey(source)
	}
	if initializerPackage, initializer, ok := owner.PackageInitializer(); ok {
		var identity strings.Builder
		appendPart(&identity, "initializer")
		appendPart(&identity, initializerPackage.Path())
		appendPart(&identity, strconv.Itoa(len(initializer.Lhs)))
		for _, variable := range initializer.Lhs {
			if !validInitializerIdentityTarget(
				initializerPackage,
				variable,
			) {
				return "", &api.NameError{
					Reason: "generated-artifact initializer owner has no semantic identity",
				}
			}
			appendPart(&identity, variable.Name())
		}
		return identity.String(), nil
	}
	return "", &api.NameError{
		Reason: "generated-artifact lexical owner is not source-reconstructible",
	}
}

func sourceArtifactOwnerKey(source types.Object) (string, error) {
	if source == nil || source.Pkg() == nil {
		return "", &api.NameError{
			Reason: "generated-artifact source owner has no package identity",
		}
	}
	var identity strings.Builder
	appendPart(&identity, source.Pkg().Path())
	switch source := source.(type) {
	case *types.Func:
		if source.Origin() != source {
			return "", &api.NameError{
				Name:   source.Name(),
				Reason: "generated-artifact source function is not its canonical origin",
			}
		}
		signature, _ := source.Type().(*types.Signature)
		if signature == nil {
			return "", &api.NameError{
				Name:   source.Name(),
				Reason: "generated-artifact source function has no signature identity",
			}
		}
		if !sourceFunctionIsExact(source, signature) {
			return "", &api.NameError{
				Name:   source.Name(),
				Reason: "generated-artifact source function is not its package declaration",
			}
		}
		appendPart(&identity, "function")
		if signature.Recv() != nil {
			receiver, ok := receiverTypeName(signature.Recv().Type())
			if !ok {
				return "", &api.NameError{
					Name:   source.Name(),
					Reason: "generated-artifact source method has no receiver identity",
				}
			}
			receiverKey, err := NamedObjectKey(receiver)
			if err != nil {
				return "", err
			}
			appendPart(&identity, receiverKey)
		}
		appendPart(&identity, source.Name())
	case *types.TypeName:
		key, err := NamedObjectKey(source)
		if err != nil {
			return "", err
		}
		appendPart(&identity, "type")
		appendPart(&identity, key)
	case *types.Const:
		if !packageObjectIsExact(source) {
			return "", &api.NameError{
				Name:   source.Name(),
				Reason: "generated-artifact constant owner is not package-owned",
			}
		}
		appendPart(&identity, "constant")
		appendPart(&identity, source.Name())
	case *types.Var:
		if !packageObjectIsExact(source) {
			return "", &api.NameError{
				Name:   source.Name(),
				Reason: "generated-artifact variable owner is not package-owned",
			}
		}
		appendPart(&identity, "variable")
		appendPart(&identity, source.Name())
	default:
		return "", &api.NameError{
			Name:   source.Name(),
			Reason: "generated-artifact source owner has no declaration identity",
		}
	}
	return identity.String(), nil
}

func SourceObjectKey(source types.Object) (string, error) {
	return sourceArtifactOwnerKey(source)
}

func sourceFunctionIsExact(
	source *types.Func,
	signature *types.Signature,
) bool {
	if source == nil || signature == nil || source.Pkg() == nil {
		return false
	}
	if signature.Recv() == nil {
		if source.Name() == "init" {
			return source.Parent() == source.Pkg().Scope() &&
				source.Pkg().Scope().Lookup(source.Name()) == nil
		}
		return packageObjectIsExact(source)
	}
	receiver, ok := receiverTypeName(signature.Recv().Type())
	if !ok {
		return false
	}
	named, ok := types.Unalias(receiver.Type()).(*types.Named)
	if !ok {
		return false
	}
	selected, _, _ := types.LookupFieldOrMethod(
		types.NewPointer(named),
		true,
		source.Pkg(),
		source.Name(),
	)
	method, ok := selected.(*types.Func)
	return ok && method.Origin() == source
}

func receiverTypeName(sourceType types.Type) (*types.TypeName, bool) {
	sourceType = types.Unalias(sourceType)
	if pointer, ok := sourceType.(*types.Pointer); ok {
		sourceType = types.Unalias(pointer.Elem())
	}
	named, ok := sourceType.(*types.Named)
	return named.Obj(), ok && named.Obj() != nil
}

func packageObjectIsExact(object types.Object) bool {
	return object != nil &&
		object.Pkg() != nil &&
		object.Parent() == object.Pkg().Scope() &&
		object.Parent().Lookup(object.Name()) == object
}

func validInitializerIdentityTarget(
	sourcePackage *types.Package,
	variable *types.Var,
) bool {
	if variable == nil || variable.Pkg() != sourcePackage {
		return false
	}
	if variable.Name() == "_" {
		return variable.Parent() == nil
	}
	return packageObjectIsExact(variable)
}

func lexicalScopePath(
	scope *types.Scope,
	root *types.Scope,
) ([]int, error) {
	var reversed []int
	for current := scope; current != root; {
		if current == nil {
			return nil, &api.NameError{
				Reason: "generated-artifact lexical scope is outside its root",
			}
		}
		parent := current.Parent()
		if parent == nil {
			return nil, &api.NameError{
				Reason: "generated-artifact lexical scope has no parent",
			}
		}
		childIndex := -1
		for index := range parent.NumChildren() {
			if parent.Child(index) == current {
				childIndex = index
				break
			}
		}
		if childIndex < 0 {
			return nil, &api.NameError{
				Reason: "generated-artifact lexical scope graph is inconsistent",
			}
		}
		reversed = append(reversed, childIndex)
		current = parent
	}
	path := make([]int, len(reversed))
	for index := range reversed {
		path[index] = reversed[len(reversed)-1-index]
	}
	return path, nil
}

func appendScopePath(target *strings.Builder, path []int) {
	appendPart(target, strconv.Itoa(len(path)))
	for _, childIndex := range path {
		appendPart(target, strconv.Itoa(childIndex))
	}
}

func objectName(object *types.TypeName) string {
	if object == nil {
		return ""
	}
	return object.Name()
}
