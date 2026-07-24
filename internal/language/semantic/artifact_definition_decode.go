package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

type wireDefinitionDecoder struct {
	identities   wireIdentityDecoder
	authority    Authority
	declarations uint64
	bindings     uint64
	initializers uint64
}

func (decoder *wireDefinitionDecoder) record(
	encoded wireDefinitionRecord,
) (DefinitionSemantics, error) {
	definition, err := decoder.identities.definition(encoded.ID)
	if err != nil {
		return DefinitionSemantics{}, err
	}
	pkg, err := decoder.identities.packageID(encoded.Package)
	if err != nil {
		return DefinitionSemantics{}, err
	}
	bindings, err := decodeReferenceRange(
		"definition bindings",
		encoded.Bindings,
		&decoder.bindings,
		decoder.identities.binding,
	)
	if err != nil {
		return DefinitionSemantics{}, err
	}
	form := DefinitionForm(encoded.Form)
	payload := encoded.Payload
	if err := requireSinglePayload(
		"definition",
		payload.Tag,
		encoded.Form,
		payload.Callable != nil,
		payload.Initializer != nil,
		payload.Bodyless != nil,
		payload.Implicit != nil,
		payload.Synthetic != nil,
	); err != nil {
		return DefinitionSemantics{}, err
	}
	spec := DefinitionSemanticsSpec{
		Definition: definition,
		Package:    pkg,
		Form:       form,
		Authority:  decoder.authority,
		Name:       encoded.Name,
		Bindings:   bindings,
	}
	switch form {
	case DefinitionFormCallable:
		if payload.Callable == nil {
			return DefinitionSemantics{}, wrongPayload(
				"definition", form,
			)
		}
		spec.Declarations, err = decodeReferenceRange(
			"callable declarations",
			payload.Callable.Declarations,
			&decoder.declarations,
			decoder.identities.declaration,
		)
		if err != nil {
			return DefinitionSemantics{}, err
		}
		spec.Signature, err = decoder.identities.typeID(
			payload.Callable.Signature,
		)
		if err != nil {
			return DefinitionSemantics{}, err
		}
		spec.Receiver, err = decoder.identities.binding(
			payload.Callable.Receiver,
		)
	case DefinitionFormInitializer:
		if payload.Initializer == nil {
			return DefinitionSemantics{}, wrongPayload(
				"definition", form,
			)
		}
		spec.Declarations, err = decodeReferenceRange(
			"initializer declarations",
			payload.Initializer.Declarations,
			&decoder.declarations,
			decoder.identities.declaration,
		)
		if err != nil {
			return DefinitionSemantics{}, err
		}
		spec.InitializerEntries, err = decodeReferenceRange(
			"initializer entries",
			payload.Initializer.Entries,
			&decoder.initializers,
			decoder.identities.occurrence,
		)
	case DefinitionFormBodyless:
		if payload.Bodyless == nil {
			return DefinitionSemantics{}, wrongPayload(
				"definition", form,
			)
		}
		declaration, decodeErr := decoder.identities.declaration(
			payload.Bodyless.Declaration,
		)
		if decodeErr != nil {
			return DefinitionSemantics{}, decodeErr
		}
		spec.Declarations = []identity.SemanticDeclarationID{
			declaration,
		}
		spec.Signature, err = decoder.identities.typeID(
			payload.Bodyless.Signature,
		)
		if err != nil {
			return DefinitionSemantics{}, err
		}
		spec.Receiver, err = decoder.identities.binding(
			payload.Bodyless.Receiver,
		)
	case DefinitionFormImplicit:
		if payload.Implicit == nil {
			return DefinitionSemantics{}, wrongPayload(
				"definition", form,
			)
		}
		spec.Implicit = identity.ImplicitDefinitionOp(
			payload.Implicit.Operation,
		)
	case DefinitionFormSynthetic:
		if payload.Synthetic == nil {
			return DefinitionSemantics{}, wrongPayload(
				"definition", form,
			)
		}
		declaration, decodeErr := decoder.identities.declaration(
			payload.Synthetic.Declaration,
		)
		if decodeErr != nil {
			return DefinitionSemantics{}, decodeErr
		}
		spec.Declarations = []identity.SemanticDeclarationID{
			declaration,
		}
		spec.Signature, err = decoder.identities.typeID(
			payload.Synthetic.Signature,
		)
	default:
		return DefinitionSemantics{}, fmt.Errorf(
			"semantic wire definition form %d is invalid",
			encoded.Form,
		)
	}
	if err != nil {
		return DefinitionSemantics{}, err
	}
	return NewDefinitionSemantics(spec)
}

func wrongPayload(name string, kind any) error {
	return fmt.Errorf(
		"semantic wire %s has no payload for %v", name, kind,
	)
}
