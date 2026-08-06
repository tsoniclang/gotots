package runtimecontract

type RuntimeFeature uint8

const (
	RuntimeFeatureInvalid   RuntimeFeature = 0
	RuntimePointerFieldPath RuntimeFeature = 1
)

func RuntimeFeatureModule(feature RuntimeFeature) (RuntimeModule, bool) {
	switch feature {
	case RuntimePointerFieldPath:
		return RuntimeModulePointer, true
	default:
		return RuntimeModuleInvalid, false
	}
}
