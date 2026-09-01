package packagestate_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func runProgram(t *testing.T, directory, name string, arguments ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOMEMLIMIT=1GiB")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(arguments, " "), err, output)
	}
	return string(output)
}

func writeProgramFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repositoryRoot() string {
	return filepath.Join("..", "..", "..", "..")
}

func isSourceFactApplication(statement tsgo.Statement) bool {
	expression, ok := statement.(tsgo.ExpressionStatement)
	if !ok {
		return false
	}
	call, ok := expression.Expression().(tsgo.CallExpression)
	if !ok {
		return false
	}
	return expressionRootedAtAttribute(call.Expression())
}

func expressionRootedAtAttribute(expression tsgo.Expression) bool {
	switch selected := expression.(type) {
	case tsgo.Identifier:
		return selected.Text() == "attribute"
	case tsgo.CallExpression:
		return expressionRootedAtAttribute(selected.Expression())
	case tsgo.PropertyAccessExpression:
		return expressionRootedAtAttribute(selected.Expression())
	default:
		return false
	}
}
