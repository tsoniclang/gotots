package memory

import "github.com/tsoniclang/gotots/internal/emit/api"

type Owner struct {
	api.Values
	api.ContainerStorageValues
	children api.ChildEmitter
}

func NewOwner(context api.Context, children api.ChildEmitter) Owner {
	return Owner{
		Values:                 context.Values(),
		ContainerStorageValues: context.ContainerStorage(),
		children:               children,
	}
}
