package api

import "go/types"

type Names interface {
	Declare(types.Object) (string, error)
	Reference(types.Object) (string, error)
	ModuleExport(types.Object) (bool, error)
}
