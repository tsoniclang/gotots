package goabi

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/environment"
)

func TestSelectedGoABI(t *testing.T) {
	for _, test := range []struct {
		order  environment.ByteOrder
		width  int64
		export string
	}{
		{environment.ByteOrderLittleEndian, 4, "little32"},
		{environment.ByteOrderLittleEndian, 8, "little64"},
		{environment.ByteOrderBigEndian, 4, "big32"},
		{environment.ByteOrderBigEndian, 8, "big64"},
	} {
		layout, err := Select(test.order, test.width)
		if err != nil {
			t.Fatal(err)
		}
		export, err := layout.Export()
		if err != nil || export != test.export {
			t.Fatalf("selected ABI = %q, %v; want %q", export, err, test.export)
		}
	}
	if _, err := Select(environment.ByteOrderInvalid, 8); err == nil {
		t.Fatal("ambient byte order accepted")
	}
	if _, err := Select(environment.ByteOrderLittleEndian, 16); err == nil {
		t.Fatal("unsupported source address width accepted")
	}
	if _, err := Invalid.Export(); err == nil {
		t.Fatal("invalid layout exported")
	}
}
