package providerboundary

import "testing"

func TestCallableProfileRootProjectionIsExact(t *testing.T) {
	for _, test := range []struct {
		name      string
		certified []int
		selected  []int
		required  bool
		want      bool
	}{
		{name: "complete", certified: []int{0, 1}, selected: []int{0, 1}, want: true},
		{name: "narrow", certified: []int{0, 1}, selected: []int{0}, want: true},
		{name: "uncertified", certified: []int{0, 1}, selected: []int{0, 2}},
		{name: "required-narrow", certified: []int{0, 1}, selected: []int{0}, required: true},
		{name: "required-complete", certified: []int{0, 1}, selected: []int{0, 1}, required: true, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := profileRootsAccept(
				test.certified,
				test.selected,
				test.required,
			); got != test.want {
				t.Fatalf("root projection accepted = %t, want %t", got, test.want)
			}
		})
	}
}

func TestCallableProfileInterfaceProjectionRejectsMissingEvidence(t *testing.T) {
	certified := map[string]struct{}{"reader": {}, "error": {}}
	if !identitySetContains(certified, map[string]struct{}{"reader": {}}) {
		t.Fatal("certified interface subset was rejected")
	}
	if identitySetContains(certified, map[string]struct{}{"writer": {}}) {
		t.Fatal("uncertified interface subset was accepted")
	}
}
