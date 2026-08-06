package environment

import "go/types"

// UseError reports an invalid environment-use contract input.
type UseError struct {
	Reason string
}

func (e *UseError) Error() string {
	return "environment use: " + e.Reason
}

// UseDemand is one closed use demand joined monotonically onto a canonical
// environment declaration when a target reference or facet is selected.
// Repeated references deduplicate the definition while joining demand.
type UseDemand uint8

const (
	UseDemandInvalid UseDemand = iota
	UseDemandTypeContract
	UseDemandValue
	UseDemandCallable
	UseDemandState
	UseDemandInitializer
	UseDemandInterfaceCapability
	UseDemandCallbackCapability
	UseDemandRuntimeFacet
)

func (d UseDemand) Valid() bool {
	return d >= UseDemandTypeContract && d <= UseDemandRuntimeFacet
}

func (d UseDemand) String() string {
	switch d {
	case UseDemandTypeContract:
		return "type-contract"
	case UseDemandValue:
		return "value"
	case UseDemandCallable:
		return "callable"
	case UseDemandState:
		return "state"
	case UseDemandInitializer:
		return "initializer"
	case UseDemandInterfaceCapability:
		return "interface-capability"
	case UseDemandCallbackCapability:
		return "callback-capability"
	case UseDemandRuntimeFacet:
		return "runtime-facet"
	default:
		return "invalid"
	}
}

// ImplementationRoute is the sole implementation route settled for one
// canonical environment declaration. Provider and boundary routes are
// derived from the canonical binding evidence and schedule an environment
// declaration artifact. Intrinsic and generated-facet routes identify
// existing compiler artifact owners and never demand a provider body.
// Exactly one route settles per declaration; a conflicting route fails at
// its first observation.
type ImplementationRoute uint8

const (
	RouteInvalid ImplementationRoute = iota
	RouteProvider
	RouteIntrinsic
	RouteGeneratedFacet
	RouteBoundary
)

func (r ImplementationRoute) Valid() bool {
	return r >= RouteProvider && r <= RouteBoundary
}

// Selection reports whether the route selects an environment declaration
// artifact (provider binding or explicit boundary) rather than an existing
// compiler-owned implementation.
func (r ImplementationRoute) Selection() bool {
	return r == RouteProvider || r == RouteBoundary
}

func (r ImplementationRoute) String() string {
	switch r {
	case RouteProvider:
		return "provider"
	case RouteIntrinsic:
		return "intrinsic"
	case RouteGeneratedFacet:
		return "generated-facet"
	case RouteBoundary:
		return "boundary"
	default:
		return "invalid"
	}
}

// ImplementationObserver is implemented by the per-file name owner so
// intrinsic and generated-facet handlers can record their sole
// implementation route through the same non-optional root observer that
// selection routes use.
type ImplementationObserver interface {
	ObserveEnvironmentImplementation(
		object types.Object,
		demand UseDemand,
		route ImplementationRoute,
	) error
}
