package emit

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	storetarget "github.com/tsoniclang/gotots/internal/emit/store"
	"github.com/tsoniclang/gotots/internal/emit/value/representation"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type emitter struct {
	source  *load.Package
	factory tsgo.Factory
	names   *nameOwner
	values  api.Values
	integer api.IntegerRepresentation
	require func(types.Object) error
}

func (e *emitter) StoreTarget(
	context api.Context,
	source ast.Expr,
) (api.StoreTargetEmission, error) {
	return storetarget.Emit(context, e, source)
}

func newEmitter(
	source *load.Package,
	factory tsgo.Factory,
	registry *declarationRegistry,
	integer api.IntegerRepresentation,
	require func(types.Object) error,
) *emitter {
	var typesInfo *types.Info
	var packageScope *types.Scope
	if source != nil {
		typesInfo = source.TypesInfo()
		packageScope = source.Types().Scope()
	}
	return &emitter{
		source:  source,
		factory: factory,
		names:   newNameOwnerWithRegistry(packageScope, typesInfo, registry),
		values:  representation.Owner{},
		integer: integer,
		require: require,
	}
}

func (e *emitter) fileContext(
	sourceFile *ast.File,
	targetPath string,
) (api.Context, error) {
	names := e.names.ForFile(
		sourceFile,
		e.source.Types().Scope(),
		e.factory,
		targetPath,
		e.require,
	)
	return api.NewContext(
		api.RoleFileDeclaration,
		e.source.FileSet(),
		e.source.Types(),
		e.source.TypesInfo(),
		e.source.TypesSizes(),
		e.factory,
		names,
		e.values,
		e.integer,
	)
}
