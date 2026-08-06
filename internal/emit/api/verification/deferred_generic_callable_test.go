package api_test

import (
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestDeferredGenericCallableOwnsRecoveryPlacement(t *testing.T) {
	factory := tsgo.NewFactory()
	reference, err := NewNameReference("target")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		placement DeferredGenericRecoveryPlacement
		want      []string
	}{
		{
			name:      "source deferred entry",
			placement: DeferredGenericRecoveryFirst,
			want:      []string{"recovery", "value"},
		},
		{
			name:      "provider profile",
			placement: DeferredGenericRecoveryLast,
			want:      []string{"value", "recovery"},
		},
		{
			name:      "provider kernel",
			placement: DeferredGenericRecoveryOmitted,
			want:      []string{"value"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target, err := NewDeferredGenericCallableReference(
				reference,
				test.placement,
			)
			if err != nil {
				t.Fatal(err)
			}
			arguments, err := target.CallArguments(
				factory.Identifier("recovery"),
				[]tsgo.Expression{factory.Identifier("value")},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(arguments) != len(test.want) {
				t.Fatalf("arguments = %d, want %d", len(arguments), len(test.want))
			}
			for index, want := range test.want {
				identifier, ok := arguments[index].(tsgo.Identifier)
				if !ok || identifier.Text() != want {
					t.Fatalf("argument %d = %T, want %q", index, arguments[index], want)
				}
			}
		})
	}
}
