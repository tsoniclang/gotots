package naming

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

const (
	interfaceDemandTransition uint8 = iota + 1
	interfaceDemandAdapter
	interfaceDemandProviderBridge
)

func (r *Registry) invalidateInterfaceDemandRequests() {
	clear(r.interfaceDemandRequests)
}

func (r *Registry) cachedInterfaceDemandRequests(
	key interfaceDemandRequestKey,
) ([]api.RootRequest, bool) {
	requests, ok := r.interfaceDemandRequests[key]
	return slices.Clone(requests), ok
}

func (r *Registry) cacheInterfaceDemandRequests(
	key interfaceDemandRequestKey,
	requests []api.RootRequest,
) []api.RootRequest {
	compact := api.CombineRequests(requests)
	r.interfaceDemandRequests[key] = compact
	return slices.Clone(compact)
}
