package emit

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type nameOwner struct {
	byObject map[types.Object]string
}

func newNameOwner() *nameOwner {
	return &nameOwner{byObject: make(map[types.Object]string)}
}

func (n *nameOwner) Declare(object types.Object) (string, error) {
	if object == nil {
		return "", &api.NameError{Reason: "declaration object is nil"}
	}
	if name, ok := n.byObject[object]; ok {
		return name, nil
	}
	if object.Name() == "" {
		return "", &api.NameError{Reason: "declaration name is empty"}
	}
	n.byObject[object] = object.Name()
	return object.Name(), nil
}

func (n *nameOwner) Reference(object types.Object) (string, error) {
	if object == nil {
		return "", &api.NameError{Reason: "reference object is nil"}
	}
	name, ok := n.byObject[object]
	if !ok {
		return "", &api.NameError{Name: object.Name(), Reason: "object has no emitted declaration"}
	}
	return name, nil
}
