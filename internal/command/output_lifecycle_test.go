package command

import (
	"crypto/sha256"
	"reflect"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestPrintPlanSeversCompilationAndTargetGraphs(t *testing.T) {
	if path, ok := forbiddenPrintPlanType(reflect.TypeOf(printPlan{}), nil); ok {
		t.Fatalf("print plan retains forbidden type %s", path)
	}
	control := struct {
		Node tsgo.SourceFile
	}{}
	if _, ok := forbiddenPrintPlanType(reflect.TypeOf(control), nil); !ok {
		t.Fatal("print-plan reachability audit did not detect its TS-Go-node control")
	}
}

func TestStagedProtocolPayloadRejectsMutation(t *testing.T) {
	original := []byte("official protocol")
	file := printPlanFile{protocolHash: sha256.Sum256(original)}
	if err := file.verifyProtocolPayload(original); err != nil {
		t.Fatal(err)
	}
	mutated := append([]byte(nil), original...)
	mutated[0]++
	if err := file.verifyProtocolPayload(mutated); err == nil {
		t.Fatal("mutated staged protocol payload was accepted")
	}
}

func forbiddenPrintPlanType(current reflect.Type, seen map[reflect.Type]struct{}) (string, bool) {
	if current == nil {
		return "", false
	}
	if seen == nil {
		seen = make(map[reflect.Type]struct{})
	}
	if _, exists := seen[current]; exists {
		return "", false
	}
	seen[current] = struct{}{}
	packagePath := current.PkgPath()
	for _, forbidden := range []string{
		"go/ast",
		"go/token",
		"go/types",
		"github.com/tsoniclang/gotots/internal/load",
		"github.com/tsoniclang/gotots/internal/emit",
		"github.com/tsoniclang/gotots/internal/target/tsgo",
	} {
		if packagePath == forbidden || strings.HasPrefix(packagePath, forbidden+"/") {
			return current.String(), true
		}
	}
	switch current.Kind() {
	case reflect.Array, reflect.Chan, reflect.Pointer, reflect.Slice:
		return forbiddenPrintPlanType(current.Elem(), seen)
	case reflect.Func:
		for index := range current.NumIn() {
			if path, ok := forbiddenPrintPlanType(current.In(index), seen); ok {
				return path, true
			}
		}
		for index := range current.NumOut() {
			if path, ok := forbiddenPrintPlanType(current.Out(index), seen); ok {
				return path, true
			}
		}
	case reflect.Interface:
		for index := range current.NumMethod() {
			if path, ok := forbiddenPrintPlanType(current.Method(index).Type, seen); ok {
				return path, true
			}
		}
	case reflect.Map:
		if path, ok := forbiddenPrintPlanType(current.Key(), seen); ok {
			return path, true
		}
		return forbiddenPrintPlanType(current.Elem(), seen)
	case reflect.Struct:
		for index := range current.NumField() {
			if path, ok := forbiddenPrintPlanType(current.Field(index).Type, seen); ok {
				return path, true
			}
		}
	}
	return "", false
}
