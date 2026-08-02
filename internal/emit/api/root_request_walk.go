package api

import "slices"

type rootRequestSequence struct {
	children []RootRequest
}

type rootRequestFrame struct {
	requests []RootRequest
	index    int
}

func combineRootRequests(groups ...[]RootRequest) []RootRequest {
	rootCount := 0
	for _, group := range groups {
		rootCount += len(group)
	}
	switch rootCount {
	case 0:
		return nil
	case 1:
		for _, group := range groups {
			if len(group) != 0 {
				return slices.Clone(group)
			}
		}
		panic("non-empty root request group disappeared")
	}

	children := make([]RootRequest, 0, rootCount)
	for _, group := range groups {
		children = append(children, group...)
	}
	return []RootRequest{{
		sequence: &rootRequestSequence{children: children},
	}}
}

func WalkRootRequests(
	requests []RootRequest,
	visit func(RootRequest) error,
) error {
	return walkRootRequestPayloads(requests, false, visit)
}

// WalkUniqueRootRequestPayloads visits each immutable payload and persistent
// sequence node once, preserving the order of their first occurrence.
func WalkUniqueRootRequestPayloads(
	requests []RootRequest,
	visit func(RootRequest) error,
) error {
	return walkRootRequestPayloads(requests, true, visit)
}

func walkRootRequestPayloads(
	requests []RootRequest,
	unique bool,
	visit func(RootRequest) error,
) error {
	if visit == nil {
		return &RootRequestError{Reason: "root request visitor is nil"}
	}
	frames := []rootRequestFrame{{requests: requests}}
	var visitedSequences map[*rootRequestSequence]struct{}
	var visitedPayloads map[*rootRequestPayload]struct{}
	if unique {
		visitedSequences = make(map[*rootRequestSequence]struct{})
		visitedPayloads = make(map[*rootRequestPayload]struct{})
	}
	for len(frames) != 0 {
		frame := &frames[len(frames)-1]
		if frame.index == len(frame.requests) {
			frames = frames[:len(frames)-1]
			continue
		}
		request := frame.requests[frame.index]
		frame.index++
		if request.sequence != nil {
			if len(request.sequence.children) == 0 {
				return &RootRequestError{
					Reason: "root request sequence is empty",
				}
			}
			if _, visited := visitedSequences[request.sequence]; visited {
				continue
			}
			if unique {
				visitedSequences[request.sequence] = struct{}{}
			}
			frames = append(frames, rootRequestFrame{
				requests: request.sequence.children,
			})
			continue
		}
		if request.Kind() == RootRequestInvalid {
			return &RootRequestError{Reason: "root request is invalid"}
		}
		if _, visited := visitedPayloads[request.payload]; visited {
			continue
		}
		if unique {
			visitedPayloads[request.payload] = struct{}{}
		}
		if err := visit(request); err != nil {
			return err
		}
	}
	return nil
}
