package integer_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestCanonicalNarrowIntegerMutationBattery(t *testing.T) {
	emission := compileIntegerFamily(
		t,
		loadIntegerFamily(t),
		integerOptions(emit.IntegerRepresentationBigInt),
		narrowOverflowRoots...,
	)
	goOutput := executeNarrowOverflowGo(t, t.TempDir())
	for _, mutation := range []struct {
		name  string
		apply func(*testing.T, tsgo.SourceFile) tsgo.SourceFile
	}{
		{"direct-multiplication", mutateUnsignedMultiplication},
		{"omitted-result-normalization", mutateSmallAddition},
		{"direct-increment", mutateSignedIncrement},
		{"direct-compound-update", mutateUnsignedCompoundUpdate},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			workingDirectory := t.TempDir()
			mutated := false
			artifacts := materializeIntegerFamilyWithTransform(
				t,
				emission,
				workingDirectory,
				func(file emit.TargetFile) tsgo.SourceFile {
					if file.Kind() != emit.TargetFileSource ||
						file.PackageName() != "integerfamily" {
						return file.SourceFile()
					}
					mutated = true
					return mutation.apply(t, file.SourceFile())
				},
			)
			if !mutated {
				t.Fatal("canonical narrow integer artifact was not mutated")
			}
			targetOutput := executeNarrowOverflowArtifacts(
				t,
				artifacts,
				workingDirectory,
			)
			if targetOutput == goOutput {
				t.Fatalf("%s mutation preserved Go behavior", mutation.name)
			}
		})
	}
}

func mutateUnsignedMultiplication(
	t *testing.T,
	source tsgo.SourceFile,
) tsgo.SourceFile {
	return mutateNarrowFunction(
		t,
		source,
		"NarrowOverflowBinary",
		func(factory tsgo.Factory, statements []tsgo.Statement) bool {
			return mutateReturnedElement(
				t,
				factory,
				statements,
				8,
				func(expression tsgo.Expression) tsgo.Expression {
					normalized, ok := expression.(tsgo.BinaryExpression)
					if !ok ||
						normalized.OperatorToken().Kind() !=
							tsgo.SyntaxKindGreaterThanGreaterThanGreaterThanToken {
						t.Fatalf("unsigned multiplication = %T, want uint32 normalization", expression)
					}
					call, ok := normalized.Left().(tsgo.CallExpression)
					if !ok || len(call.Arguments()) != 2 {
						t.Fatalf("unsigned multiplication = %T, want Math.imul call", normalized.Left())
					}
					member, ok := call.Expression().(tsgo.PropertyAccessExpression)
					if !ok {
						t.Fatalf("unsigned multiplication calls %T, want Math.imul", call.Expression())
					}
					memberName, nameOK := member.Name().(tsgo.Identifier)
					if !nameOK || memberName.Text() != "imul" {
						t.Fatalf("unsigned multiplication calls %T, want Math.imul", call.Expression())
					}
					arguments := call.Arguments()
					direct := factory.BinaryExpression(
						nil,
						arguments[0],
						nil,
						factory.BinaryOperatorToken(tsgo.BinaryOperatorAsteriskToken),
						arguments[1],
					)
					return factory.BinaryExpression(
						nil,
						direct,
						nil,
						normalized.OperatorToken(),
						normalized.Right(),
					)
				},
			)
		},
	)
}

func mutateSmallAddition(
	t *testing.T,
	source tsgo.SourceFile,
) tsgo.SourceFile {
	return mutateNarrowFunction(
		t,
		source,
		"NarrowOverflowBinary",
		func(factory tsgo.Factory, statements []tsgo.Statement) bool {
			return mutateReturnedElement(
				t,
				factory,
				statements,
				5,
				func(expression tsgo.Expression) tsgo.Expression {
					rightShift, ok := expression.(tsgo.BinaryExpression)
					if !ok ||
						rightShift.OperatorToken().Kind() !=
							tsgo.SyntaxKindGreaterThanGreaterThanToken {
						t.Fatalf("int8 addition = %T, want signed normalization", expression)
					}
					leftShift, ok := rightShift.Left().(tsgo.BinaryExpression)
					if !ok ||
						leftShift.OperatorToken().Kind() !=
							tsgo.SyntaxKindLessThanLessThanToken {
						t.Fatalf("int8 addition normalization = %T, want left shift", rightShift.Left())
					}
					return leftShift.Left()
				},
			)
		},
	)
}

func mutateSignedIncrement(
	t *testing.T,
	source tsgo.SourceFile,
) tsgo.SourceFile {
	return mutateNarrowFunction(
		t,
		source,
		"NarrowOverflowUpdate",
		func(factory tsgo.Factory, statements []tsgo.Statement) bool {
			return replaceNormalizedUpdate(
				t,
				factory,
				statements,
				"maxSigned",
				func(left tsgo.Expression) tsgo.Expression {
					return factory.PostfixUnaryExpression(
						left,
						tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
					)
				},
			)
		},
	)
}

func mutateUnsignedCompoundUpdate(
	t *testing.T,
	source tsgo.SourceFile,
) tsgo.SourceFile {
	return mutateNarrowFunction(
		t,
		source,
		"NarrowOverflowUpdate",
		func(factory tsgo.Factory, statements []tsgo.Statement) bool {
			return replaceNormalizedUpdate(
				t,
				factory,
				statements,
				"maxUnsigned",
				func(left tsgo.Expression) tsgo.Expression {
					return factory.BinaryExpression(
						nil,
						left,
						nil,
						factory.BinaryOperatorToken(
							tsgo.BinaryOperatorPlusEqualsToken,
						),
						factory.NumericLiteral("1", tsgo.TokenFlagsNone),
					)
				},
			)
		},
	)
}

func mutateNarrowFunction(
	t *testing.T,
	source tsgo.SourceFile,
	functionName string,
	mutate func(tsgo.Factory, []tsgo.Statement) bool,
) tsgo.SourceFile {
	t.Helper()
	factory := tsgo.NewFactory()
	statements := source.Statements()
	mutated := false
	for index, statement := range statements {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if !ok || function.Name().Text() != functionName {
			continue
		}
		body := function.Body().(tsgo.Block)
		bodyStatements := body.Statements()
		if !mutate(factory, bodyStatements) {
			t.Fatalf("target function %s contains no mutation site", functionName)
		}
		statements[index] = factory.FunctionDeclaration(
			function.Modifiers(),
			function.AsteriskToken(),
			function.Name(),
			function.TypeParameters(),
			function.Parameters(),
			function.Type(),
			factory.Block(bodyStatements, body.MultiLine()),
		)
		mutated = true
	}
	if !mutated {
		t.Fatalf("target function %s was not mutated", functionName)
	}
	return factory.SourceFile(
		statements,
		source.EndOfFileToken(),
		source.SourceData(),
	)
}

func mutateReturnedElement(
	t *testing.T,
	factory tsgo.Factory,
	statements []tsgo.Statement,
	elementIndex int,
	mutate func(tsgo.Expression) tsgo.Expression,
) bool {
	t.Helper()
	for index, statement := range statements {
		returned, ok := statement.(tsgo.ReturnStatement)
		if !ok {
			continue
		}
		array, ok := returned.Expression().(tsgo.ArrayLiteralExpression)
		if !ok {
			t.Fatal("narrow overflow function does not return an array")
		}
		elements := array.Elements()
		if len(elements) <= elementIndex {
			t.Fatalf("narrow overflow result count = %d, want index %d", len(elements), elementIndex)
		}
		elements[elementIndex] = mutate(elements[elementIndex])
		statements[index] = factory.ReturnStatement(
			factory.ArrayLiteralExpression(elements, array.MultiLine()),
		)
		return true
	}
	return false
}

func replaceNormalizedUpdate(
	t *testing.T,
	factory tsgo.Factory,
	statements []tsgo.Statement,
	targetName string,
	replacement func(tsgo.Expression) tsgo.Expression,
) bool {
	t.Helper()
	for index, statement := range statements {
		expressionStatement, ok := statement.(tsgo.ExpressionStatement)
		if !ok {
			continue
		}
		assignment, ok := expressionStatement.Expression().(tsgo.BinaryExpression)
		if !ok || assignment.OperatorToken().Kind() != tsgo.SyntaxKindEqualsToken {
			continue
		}
		left, ok := assignment.Left().(tsgo.Identifier)
		if !ok || left.Text() != targetName {
			continue
		}
		statements[index] = factory.ExpressionStatement(replacement(left))
		return true
	}
	return false
}
