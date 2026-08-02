package api

import "slices"

type RecoveryCallableResolver interface {
	ObserveRecoveryCallable(
		Context,
		CallableFacet,
	) (RecoveryCallableObservation, error)
}

type RecoveryCallableObservation struct {
	recovery bool
	requests []RootRequest
}

func NewRecoveryCallableObservation(
	recovery bool,
	requests ...RootRequest,
) (RecoveryCallableObservation, error) {
	if err := validateReferenceRequests(requests); err != nil {
		return RecoveryCallableObservation{}, &RootRequestError{
			Reason: "recovery callable observation has an invalid request",
		}
	}
	return RecoveryCallableObservation{
		recovery: recovery,
		requests: slices.Clone(requests),
	}, nil
}

func (o RecoveryCallableObservation) Recovery() bool {
	return o.recovery
}

func (o RecoveryCallableObservation) Requests() []RootRequest {
	return slices.Clone(o.requests)
}

func (c Context) WithRecoveryCallableResolver(
	resolver RecoveryCallableResolver,
) Context {
	if resolver == nil {
		panic("recovery callable resolver is nil")
	}
	c.recoveryResolver = resolver
	return c
}

func (c Context) ObserveRecoveryCallable(
	facet CallableFacet,
) (RecoveryCallableObservation, error) {
	if c.recoveryResolver == nil {
		return RecoveryCallableObservation{}, &ContextError{
			Reason: "recovery callable resolver is unavailable",
		}
	}
	if !facet.Valid() {
		return RecoveryCallableObservation{}, &ContextError{
			Reason: "recovery callable facet is invalid",
		}
	}
	if !c.artifactOwner.Valid() {
		return RecoveryCallableObservation{}, &ContextError{
			Reason: "recovery callable consumer has no artifact owner",
		}
	}
	return c.recoveryResolver.ObserveRecoveryCallable(c, facet)
}
