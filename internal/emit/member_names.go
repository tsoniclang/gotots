package emit

import (
	"go/types"
	"slices"
	"strconv"
)

func (n *nameOwner) preallocateMethods(info *types.Info) {
	var methods []*types.Func
	for _, object := range info.Defs {
		method, ok := object.(*types.Func)
		if !ok {
			continue
		}
		signature, ok := method.Type().(*types.Signature)
		if !ok || signature.Recv() == nil {
			continue
		}
		methods = append(methods, method)
	}
	slices.SortFunc(methods, func(left, right *types.Func) int {
		return compareNameObjects(left, right)
	})
	used := make(map[string]struct{})
	for _, name := range n.targetNameByObject {
		used[name] = struct{}{}
	}
	for _, method := range methods {
		signature := method.Type().(*types.Signature)
		receiverName := receiverTypeName(signature.Recv().Type())
		base := portableIdentifier(receiverName) + "_" +
			portableIdentifier(method.Name())
		candidate := base
		for suffix := uint64(1); ; suffix++ {
			_, targetCollision := used[candidate]
			_, sourceCollision := n.sourceNameBases[candidate]
			if !targetCollision && !sourceCollision {
				break
			}
			candidate = base + "__method_" + strconv.FormatUint(suffix, 10)
		}
		used[candidate] = struct{}{}
		n.targetNameByObject[method] = candidate
	}
}

func receiverTypeName(sourceType types.Type) string {
	if pointer, ok := types.Unalias(sourceType).(*types.Pointer); ok {
		sourceType = pointer.Elem()
	}
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Obj() == nil {
		return "receiver"
	}
	return named.Obj().Name()
}

func (n *nameOwner) preallocateMembers(packageScope *types.Scope) {
	names := packageScope.Names()
	slices.Sort(names)
	for _, name := range names {
		typeName, ok := packageScope.Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := types.Unalias(typeName.Type()).(*types.Named)
		if !ok {
			continue
		}
		structType, ok := named.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		used := map[string]struct{}{
			"constructor": {},
		}
		for index := range structType.NumFields() {
			field := structType.Field(index)
			base := portableIdentifier(field.Name())
			candidate := base
			for suffix := uint64(1); ; suffix++ {
				if _, duplicate := used[candidate]; !duplicate {
					break
				}
				candidate = base + "__field_" + strconv.FormatUint(suffix, 10)
			}
			used[candidate] = struct{}{}
			n.memberNameByObject[field] = candidate
		}
	}
}
