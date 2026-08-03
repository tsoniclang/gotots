package gostdlib

type ProviderCallableParameterDocument struct {
	Parameter int        `json:"parameter"`
	Effect    EffectKind `json:"effect"`
}
