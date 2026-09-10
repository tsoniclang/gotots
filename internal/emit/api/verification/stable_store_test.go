package api_test

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type stableStoreServices struct {
	api.Names
	api.Values
	temporaries int
	writes      int
	projections int
}

func (services *stableStoreServices) Temporary(api.TemporaryKind) (string, error) {
	services.temporaries++
	return fmt.Sprintf("captured%d", services.temporaries), nil
}

func (services *stableStoreServices) AssignStable(context api.Context, _ ast.Node, _ types.Type, _ tsgo.Expression, _ api.ExpressionEmission) (api.ExpressionEmission, error) {
	services.writes++
	return api.DirectExpression(context.Factory().Identifier("stableWrite")), nil
}

func (services *stableStoreServices) FromStorage(_ api.Context, _ ast.Node, _ types.Type, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	services.projections++
	return value, nil
}

func TestStableStoreSurvivesLocationCapture(test *testing.T) {
	factory := tsgo.NewFactory()
	array := types.NewArray(types.Typ[types.Int32], 2)
	receiver := api.DirectExpression(factory.Identifier("holder"))
	constructors := []struct {
		name       string
		projection int
		create     func() (api.StoreTargetEmission, error)
	}{
		{"variable", 0, func() (api.StoreTargetEmission, error) {
			return api.NewStoreTargetEmission(factory.Identifier("value"), array, nil)
		}},
		{"property", 0, func() (api.StoreTargetEmission, error) {
			return api.NewPropertyStoreTargetEmission(factory, receiver, "value", array)
		}},
		{"accessor", 0, func() (api.StoreTargetEmission, error) {
			return api.NewAccessorStoreTargetEmission(receiver, "get", "set", nil, array)
		}},
		{"canonical-accessor", 1, func() (api.StoreTargetEmission, error) {
			return api.NewCanonicalStorageAccessorStoreTargetEmission(receiver, "get", "set", nil, array)
		}},
		{"pointer", 0, func() (api.StoreTargetEmission, error) {
			return api.NewFunctionStoreTargetEmission(api.DirectExpression(factory.Identifier("load")), api.DirectExpression(factory.Identifier("store")), []api.ExpressionEmission{receiver}, array)
		}},
	}
	for _, constructor := range constructors {
		test.Run(constructor.name, func(test *testing.T) {
			services := &stableStoreServices{}
			context, err := api.NewContext(api.RoleAssignmentTarget, token.NewFileSet(), types.NewPackage("example.com/store", "store"), &types.Info{}, types.SizesFor("gc", "amd64"), api.MemoryByteOrderLittleEndian, factory, services, services, api.IntegerRepresentationNumber, api.EvaluationOrderPreserveGo)
			if err != nil {
				test.Fatal(err)
			}
			target, err := constructor.create()
			if err != nil {
				test.Fatal(err)
			}
			target, err = target.WithStableIdentity()
			if err != nil {
				test.Fatal(err)
			}
			captured, err := target.CaptureLocation(context)
			if err != nil {
				test.Fatal(err)
			}
			result, err := captured.StoreValue(context, nil, api.DirectExpression(factory.Identifier("replacement")))
			if err != nil {
				test.Fatal(err)
			}
			if services.writes != 1 || services.projections != constructor.projection || services.temporaries != 1 {
				test.Fatalf("writes=%d projections=%d captures=%d", services.writes, services.projections, services.temporaries)
			}
			if result.Value().(tsgo.Identifier).Text() != "stableWrite" || len(result.Before()) != 1 {
				test.Fatal("captured target lost its stable write or prerequisite")
			}
		})
	}
}

func TestStableStoreRejectsNonAddressableOrNonAggregateTargets(test *testing.T) {
	factory := tsgo.NewFactory()
	copying, err := api.NewCopyingAccessorStoreTargetEmission(api.DirectExpression(factory.Identifier("map")), "get", "set", nil, types.NewArray(types.Typ[types.Int32], 2))
	if err != nil {
		test.Fatal(err)
	}
	scalar, err := api.NewStoreTargetEmission(factory.Identifier("scalar"), types.Typ[types.Int32], nil)
	if err != nil {
		test.Fatal(err)
	}
	for _, target := range []api.StoreTargetEmission{{}, copying, scalar} {
		if _, err := target.WithStableIdentity(); err == nil {
			test.Fatal("invalid stable target was admitted")
		}
	}
}
