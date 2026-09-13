package runtime

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestLifetimeRuntimeHasOneExactCallableOwner(test *testing.T) {
	factory := tsgo.NewFactory()
	definitions, err := Build(factory, api.RuntimeModuleLifetime, []api.RuntimeSymbol{api.RuntimeKeepAlive})
	if err != nil {
		test.Fatal(err)
	}
	if len(definitions) != 1 || definitions[0].Symbol() != api.RuntimeKeepAlive {
		test.Fatal("lifetime runtime did not produce its exact definition")
	}
	if _, valid := definitions[0].Statement().(tsgo.FunctionDeclaration); !valid {
		test.Fatal("lifetime definition is not a callable")
	}
	for _, symbols := range [][]api.RuntimeSymbol{
		{api.RuntimeKeepAlive, api.RuntimeKeepAlive},
		{api.RuntimeStringValue},
	} {
		if _, err := Build(factory, api.RuntimeModuleLifetime, symbols); err == nil {
			test.Fatal("lifetime module accepted duplicated or foreign ownership")
		}
	}
}
