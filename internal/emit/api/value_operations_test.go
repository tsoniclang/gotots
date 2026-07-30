package api

import (
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestAccessorReadPreservesReceiverPrerequisitesAndRequests(t *testing.T) {
	factory := tsgo.NewFactory()
	request, err := NewImportRequest(
		factory,
		ImportPhaseValue,
		"./box.js",
		"Box",
		"Box",
	)
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := NewExpressionEmission(
		[]tsgo.Statement{
			factory.ExpressionStatement(factory.Identifier("prepare")),
		},
		factory.Identifier("Box"),
		[]RootRequest{request},
	)
	if err != nil {
		t.Fatal(err)
	}
	target, err := NewAccessorStoreTargetEmission(
		receiver,
		"$get",
		"$set",
		nil,
		types.Typ[types.Int32],
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := target.ReadValue(Context{factory: factory}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Before()) != 1 ||
		result.Before()[0].(tsgo.ExpressionStatement).
			Expression().(tsgo.Identifier).Text() != "prepare" ||
		len(result.Requests()) != 1 ||
		result.Requests()[0].ExportedName() != "Box" {
		t.Fatalf(
			"accessor read prerequisites = %d, requests = %d",
			len(result.Before()),
			len(result.Requests()),
		)
	}
}
