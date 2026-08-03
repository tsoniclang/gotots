package gostdlib

import "fmt"

type ProviderCallableParameterDocument struct {
	Parameter int        `json:"parameter"`
	Effect    EffectKind `json:"effect"`
}

func validateCallableParameters(
	parameters []ProviderCallableParameterDocument,
	field string,
	allowAsynchronous bool,
) error {
	previous := -1
	for index, selected := range parameters {
		if selected.Parameter < 0 || selected.Parameter <= previous {
			return manifestError(
				field,
				"parameters are negative, duplicated, or not strictly ordered",
			)
		}
		validEffect := selected.Effect == EffectSynchronous ||
			selected.Effect == EffectAwaitable ||
			allowAsynchronous && selected.Effect == EffectAsynchronous
		if !validEffect {
			return manifestError(
				fmt.Sprintf("%s[%d].effect", field, index),
				"value is invalid",
			)
		}
		previous = selected.Parameter
	}
	return nil
}
