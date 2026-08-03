package environment_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/environment"
)

func TestBuildProfileByteOrderIsClosedOverSupportedArchitectures(t *testing.T) {
	tests := []struct {
		architecture string
		want         environment.ByteOrder
	}{
		{"386", environment.ByteOrderLittleEndian},
		{"amd64", environment.ByteOrderLittleEndian},
		{"arm", environment.ByteOrderLittleEndian},
		{"arm64", environment.ByteOrderLittleEndian},
		{"loong64", environment.ByteOrderLittleEndian},
		{"mips", environment.ByteOrderBigEndian},
		{"mips64", environment.ByteOrderBigEndian},
		{"mips64le", environment.ByteOrderLittleEndian},
		{"mipsle", environment.ByteOrderLittleEndian},
		{"ppc64", environment.ByteOrderBigEndian},
		{"ppc64le", environment.ByteOrderLittleEndian},
		{"riscv64", environment.ByteOrderLittleEndian},
		{"s390x", environment.ByteOrderBigEndian},
		{"wasm", environment.ByteOrderLittleEndian},
	}
	for _, test := range tests {
		profile, err := environment.NewBuildProfileForToolchain(
			"go1.26.4",
			"linux",
			test.architecture,
			false,
			nil,
		)
		if err != nil {
			t.Fatalf("profile %s: %v", test.architecture, err)
		}
		got, err := profile.ByteOrder()
		if err != nil {
			t.Fatalf("byte order %s: %v", test.architecture, err)
		}
		if got != test.want {
			t.Fatalf("byte order %s = %d, want %d", test.architecture, got, test.want)
		}
	}
}

func TestBuildProfileByteOrderRejectsUnknownArchitecture(t *testing.T) {
	profile, err := environment.NewBuildProfileForToolchain(
		"go1.26.4",
		"linux",
		"futurearch",
		false,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if order, err := profile.ByteOrder(); err == nil || order.Valid() {
		t.Fatalf("unknown architecture byte order = %d, %v", order, err)
	}
}
