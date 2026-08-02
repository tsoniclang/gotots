package emit_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestBuiltinErrorUsesSelectedCooperativeContract(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/cooperativeerror\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package cooperativeerror

import "errors"

type waitingError struct {
	ready <-chan string
}

func (failure *waitingError) Error() string {
	return <-failure.ready
}

type plainError string

func (failure plainError) Error() string {
	return string(failure)
}

type Lookalike interface {
	Error() string
}

func Waiting(ready <-chan string) error {
	return &waitingError{ready: ready}
}

func Plain() error {
	return plainError("plain")
}

func Provider() error {
	return errors.New("provider")
}

func LookalikeMessage(failure Lookalike) string {
	return failure.Error()
}

func Message(failure error) string {
	return failure.Error()
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if !strings.Contains(
		artifacts.printed,
		"Error(): Promise<gostring>;",
	) {
		t.Fatalf(
			"canonical error contract is not Go-shaped and cooperative:\n%s",
			artifacts.printed,
		)
	}
	if count := strings.Count(
		artifacts.printed,
		"async Error(): Promise<gostring>",
	); count < 2 {
		t.Fatalf(
			"source error adapters did not converge on one Go-shaped cooperative contract: count=%d\n%s",
			count,
			artifacts.printed,
		)
	}
	if strings.Contains(artifacts.printed, "Error$deferred") {
		t.Fatalf(
			"non-recovering error methods acquired a private recovery entry:\n%s",
			artifacts.printed,
		)
	}
	if strings.Contains(artifacts.printed, "Error($go$recovery") {
		t.Fatalf(
			"source error contract exposes recovery authority:\n%s",
			artifacts.printed,
		)
	}
	if !strings.Contains(artifacts.printed, "export interface Lookalike") {
		t.Fatalf(
			"source-declared lookalike interface lost its declaration identity:\n%s",
			artifacts.printed,
		)
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
}
