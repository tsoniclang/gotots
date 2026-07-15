package ir

import (
	"go/types"
)

// admitGenericFunction verifies a generic function declaration under the
// unit's closed-world instantiation evidence and returns its type
// parameter names. A single generic body is exact only when no recorded
// instantiation binds a type parameter to a struct value (whose copy
// semantics the body cannot express) or to a type outside the reviewed
// subset; the opaque type-parameter kind then admits only operations
// whose lowering is identical across every remaining carrier.
func (b *builder) admitGenericFunction(object *types.Func, span Span) ([]string, error) {
	signature := object.Type().(*types.Signature)
	typeParams := signature.TypeParams()
	names := make([]string, 0, typeParams.Len())
	for i := range typeParams.Len() {
		names = append(names, typeParams.At(i).Obj().Name())
	}

	for _, instance := range b.unit.GenericInstances(object) {
		for _, arg := range instance {
			resolved, err := b.typeOf(arg, span)
			if err != nil {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "generic function instantiated with an unreviewed type argument (" + arg.String() + ")", Span: span}
			}
			if resolved.Kind == KindStruct {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "generic function instantiated with a struct value (copy semantics vary per instantiation)", Span: span}
			}
		}
	}
	return names, nil
}
