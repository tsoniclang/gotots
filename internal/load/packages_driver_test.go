package load

import (
	"strings"
	"testing"
)

func TestDriverRequestRejectsMalformedTrailingInput(t *testing.T) {
	if _, err := decodeDriverRequest(strings.NewReader(`{"mode":1}`)); err != nil {
		t.Fatalf("valid driver request was rejected: %v", err)
	}
	if _, err := decodeDriverRequest(strings.NewReader(`{"mode":1} trailing`)); err == nil {
		t.Fatal("malformed trailing driver input was accepted")
	}
	if _, err := decodeDriverRequest(strings.NewReader(`{"mode":1}{"mode":1}`)); err == nil {
		t.Fatal("second driver request was accepted")
	}
}
