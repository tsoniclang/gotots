package api

import "slices"

const (
	invalidRequestKindMask        = uint8(1) << RootRequestInvalid
	declarationRequestKindMask    = uint8(1) << RootRequestDeclarationRequirement
	nonDeclarationRequestKindMask = uint8(1)<<RootRequestImport |
		uint8(1)<<RootRequestArtifactDependency
)

type rootRequestSelection struct {
	request  RootRequest
	selected bool
}

func (r RootRequest) NestedRequests() ([]RootRequest, bool) {
	if r.sequence == nil {
		return nil, false
	}
	return slices.Clone(r.sequence.children), true
}

func SelectDeclarationRequests(
	requests []RootRequest,
) ([]RootRequest, error) {
	return selectRootRequests(requests, declarationRequestKindMask)
}

func SelectNonDeclarationRequests(
	requests []RootRequest,
) ([]RootRequest, error) {
	return selectRootRequests(requests, nonDeclarationRequestKindMask)
}

func selectRootRequests(
	requests []RootRequest,
	selectedKinds uint8,
) ([]RootRequest, error) {
	selected, _, err := selectRootRequestsWithWork(requests, selectedKinds)
	return selected, err
}

func selectRootRequestsWithWork(
	requests []RootRequest,
	selectedKinds uint8,
) ([]RootRequest, uint64, error) {
	memo := make(map[*rootRequestSequence]rootRequestSelection)
	selected := make([]RootRequest, 0, len(requests))
	var work uint64
	for _, request := range requests {
		selection, err := selectRootRequest(
			request,
			selectedKinds,
			memo,
			&work,
		)
		if err != nil {
			return nil, work, err
		}
		if selection.selected {
			selected = append(selected, selection.request)
		}
	}
	return slices.Clone(selected), work, nil
}

func selectRootRequest(
	request RootRequest,
	selectedKinds uint8,
	memo map[*rootRequestSequence]rootRequestSelection,
	work *uint64,
) (rootRequestSelection, error) {
	*work++
	if request.sequence == nil {
		if request.Kind() == RootRequestInvalid {
			return rootRequestSelection{}, &RootRequestError{
				Reason: "root request is invalid",
			}
		}
		return rootRequestSelection{
			request:  request,
			selected: request.rootRequestKinds()&selectedKinds != 0,
		}, nil
	}
	if selection, ok := memo[request.sequence]; ok {
		return selection, nil
	}
	if len(request.sequence.children) == 0 {
		return rootRequestSelection{}, &RootRequestError{
			Reason: "root request sequence is empty",
		}
	}
	if request.sequence.kinds&invalidRequestKindMask == 0 &&
		request.sequence.kinds&^selectedKinds == 0 {
		selection := rootRequestSelection{request: request, selected: true}
		memo[request.sequence] = selection
		return selection, nil
	}
	if request.sequence.kinds&selectedKinds == 0 &&
		request.sequence.kinds&invalidRequestKindMask == 0 {
		memo[request.sequence] = rootRequestSelection{}
		return rootRequestSelection{}, nil
	}
	children := make([]RootRequest, 0, len(request.sequence.children))
	unchanged := true
	for _, child := range request.sequence.children {
		selection, err := selectRootRequest(
			child,
			selectedKinds,
			memo,
			work,
		)
		if err != nil {
			return rootRequestSelection{}, err
		}
		if !selection.selected {
			unchanged = false
			continue
		}
		children = append(children, selection.request)
		if selection.request != child {
			unchanged = false
		}
	}
	selection := rootRequestSelection{}
	switch {
	case len(children) == 0:
	case unchanged && len(children) == len(request.sequence.children):
		selection = rootRequestSelection{request: request, selected: true}
	default:
		combined := combineRootRequests(children)
		selection = rootRequestSelection{
			request:  combined[0],
			selected: true,
		}
	}
	memo[request.sequence] = selection
	return selection, nil
}

func (r RootRequest) rootRequestKinds() uint8 {
	if r.sequence != nil {
		return r.sequence.kinds
	}
	return uint8(1) << r.Kind()
}

func NewDeclarationRequirementRequest(
	requirement DeclarationRequirement,
) (RootRequest, error) {
	if !requirement.Valid() {
		return RootRequest{}, &RootRequestError{
			Reason: "declaration requirement is invalid",
		}
	}
	return newDeclarationRequirementRequest(requirement), nil
}
