package providerboundary

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
)

func RequireSynchronousCallable(
	context api.Context,
	owner *types.Func,
) error {
	if context.ConcurrencySemantics() != api.ConcurrencySemanticsDisabled {
		return nil
	}
	names, ok := context.Names().(api.ProviderCallableEffectNames)
	if !ok {
		return boundaryInvariant(
			context,
			"provider callable-effect evidence is unavailable",
		)
	}
	effect, providerOwned, err := names.ProviderCallableEffect(owner)
	if err != nil || !providerOwned {
		return err
	}
	return RequireSynchronousEffect(context, owner.FullName(), effect)
}

func RequireSynchronousEffect(
	context api.Context,
	identity string,
	effect gostdlib.EffectKind,
) error {
	if context.ConcurrencySemantics() != api.ConcurrencySemanticsDisabled {
		return nil
	}
	if !effect.Valid() {
		return boundaryInvariant(
			context,
			"selected provider callable has invalid effect evidence for "+identity,
		)
	}
	return RequireSynchronousSuspension(
		context,
		identity,
		effect.MaySuspend(),
	)
}

func RequireSynchronousSuspension(
	context api.Context,
	identity string,
	maySuspend bool,
) error {
	if context.ConcurrencySemantics() != api.ConcurrencySemanticsDisabled {
		return nil
	}
	if maySuspend {
		return boundaryInvariant(
			context,
			"disabled concurrency selected a suspending provider callable for "+identity,
		)
	}
	return nil
}

func RequireProviderDefinedCallableInput(
	context api.Context,
	model definedtype.Model,
	synchronous bool,
) error {
	effect, providerOwned, err := providerDefinedCallableEffect(context, model)
	if err != nil || !providerOwned {
		return err
	}
	expected := gostdlib.EffectAwaitable
	if synchronous ||
		context.ConcurrencySemantics() == api.ConcurrencySemanticsDisabled {
		expected = gostdlib.EffectSynchronous
	}
	if effect != expected {
		return boundaryInvariant(
			context,
			"provider defined-callable input effect is "+string(effect)+
				", want "+string(expected)+" for "+model.TypeName().Name(),
		)
	}
	return nil
}

func RequireProviderDefinedCallableOutput(
	context api.Context,
	model definedtype.Model,
) error {
	effect, providerOwned, err := providerDefinedCallableEffect(context, model)
	if err != nil || !providerOwned {
		return err
	}
	if context.ConcurrencySemantics() == api.ConcurrencySemanticsDisabled &&
		effect != gostdlib.EffectSynchronous {
		return boundaryInvariant(
			context,
			"disabled concurrency selected a suspending provider defined callable for "+
				model.TypeName().Name(),
		)
	}
	return nil
}

func providerDefinedCallableEffect(
	context api.Context,
	model definedtype.Model,
) (gostdlib.EffectKind, bool, error) {
	if model.TypeName() == nil {
		return gostdlib.EffectInvalid, false, nil
	}
	representation, err := model.Representation(context)
	if err != nil {
		return gostdlib.EffectInvalid, false, err
	}
	switch representation.Kind() {
	case api.DefinedValueRepresentationProviderCanonical,
		api.DefinedValueRepresentationProviderOperations:
	case api.DefinedValueRepresentationGeneratedWrapper,
		api.DefinedValueRepresentationGeneratedNumeric:
		return gostdlib.EffectInvalid, false, nil
	default:
		return gostdlib.EffectInvalid, false, boundaryInvariant(
			context,
			"provider defined-callable representation is invalid",
		)
	}
	names, ok := context.Names().(api.ProviderDefinedCallableEffectNames)
	if !ok {
		return gostdlib.EffectInvalid, false, boundaryInvariant(
			context,
			"provider defined-callable effect evidence is unavailable",
		)
	}
	effect, providerOwned, err := names.ProviderDefinedCallableEffect(
		model.TypeName(),
	)
	if err != nil {
		return gostdlib.EffectInvalid, providerOwned, err
	}
	if !providerOwned || !effect.Valid() {
		return gostdlib.EffectInvalid, providerOwned, boundaryInvariant(
			context,
			"provider defined-callable effect evidence is absent",
		)
	}
	return effect, true, nil
}
