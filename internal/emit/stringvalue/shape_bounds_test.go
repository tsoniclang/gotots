package stringvalue_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func stringValueMethod(t *testing.T, source tsgo.SourceFile, name string) tsgo.MethodDeclaration {
	t.Helper()
	for _, statement := range source.Statements() {
		class, ok := statement.(tsgo.ClassDeclaration)
		if !ok || class.Name().Text() != "GoString" {
			continue
		}
		for _, element := range class.Members() {
			method, ok := element.(tsgo.MethodDeclaration)
			if ok && method.Name().(tsgo.Identifier).Text() == name {
				return method
			}
		}
	}
	t.Fatalf("canonical string method %s is absent", name)
	return nil
}

func hasPanicBoundsCheck(body tsgo.Block) bool {
	for _, statement := range body.Statements() {
		check, ok := statement.(tsgo.IfStatement)
		if !ok {
			continue
		}
		consequent := []tsgo.Statement{check.ThenStatement()}
		if block, ok := check.ThenStatement().(tsgo.Block); ok {
			consequent = block.Statements()
		}
		for _, nested := range consequent {
			expression, ok := nested.(tsgo.ExpressionStatement)
			if !ok {
				continue
			}
			call, ok := expression.Expression().(tsgo.CallExpression)
			if !ok {
				continue
			}
			member, ok := call.Expression().(tsgo.PropertyAccessExpression)
			if ok &&
				member.Name().(tsgo.Identifier).Text() == "raiseRuntime" {
				return true
			}
		}
	}
	return false
}
