package maprepresentation

import (
	"path/filepath"
	"runtime"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func specializationRepositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve specialization repository root")
	}
	return filepath.Clean(
		filepath.Join(filepath.Dir(file), "..", "..", "..", ".."),
	)
}

func staticField(
	context api.Context,
	value tsgo.Expression,
	name string,
) tsgo.PropertyAccessExpression {
	return context.Factory().PropertyAccessExpression(
		value,
		nil,
		context.Factory().Identifier(name),
		tsgo.NodeFlagsNone,
	)
}
