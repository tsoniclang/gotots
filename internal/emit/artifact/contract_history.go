package artifact

import (
	"bytes"
	"compress/flate"
	"io"
	"sync"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type contractFingerprint struct {
	first  uint64
	second uint64
}

type contractHistoryEntry struct {
	fingerprint contractFingerprint
	previous    contractReverseDelta
}

type contractHistory struct {
	entries []contractHistoryEntry
}

type contractReverseDelta struct {
	facets  [api.ArtifactFacetExportSurface + 1]facetReverseDelta
	changed uint8
}

type facetReverseDelta struct {
	previousPresent bool
	prefix          int
	suffix          int
	previousMiddle  exactHistoryBytes
}

type exactHistoryBytes struct {
	data       []byte
	rawLength  int
	compressed bool
}

var contractHistoryWriterPool = sync.Pool{
	New: func() any {
		writer, err := flate.NewWriter(io.Discard, flate.BestSpeed)
		if err != nil {
			panic(err)
		}
		return writer
	},
}

func (h *contractHistory) initialize(contract Contract) {
	if len(h.entries) != 0 {
		panic("artifact contract history initialized twice")
	}
	h.entries = append(h.entries, contractHistoryEntry{
		fingerprint: fingerprintContract(contract),
	})
}

func (h *contractHistory) append(previous Contract, next Contract) {
	if len(h.entries) == 0 {
		panic("artifact contract history append before initialization")
	}
	h.entries = append(h.entries, contractHistoryEntry{
		fingerprint: fingerprintContract(next),
		previous:    reverseContractDelta(previous, next),
	})
}

func (h *contractHistory) contains(
	current Contract,
	candidate Contract,
) bool {
	fingerprint := fingerprintContract(candidate)
	for index := len(h.entries) - 1; index >= 0; index-- {
		if h.entries[index].fingerprint != fingerprint {
			continue
		}
		historical := current
		for revision := len(h.entries) - 1; revision > index; revision-- {
			historical = h.entries[revision].previous.restore(historical)
		}
		if equalArtifactContracts(historical, candidate) {
			return true
		}
	}
	return false
}

func reverseContractDelta(
	previous Contract,
	next Contract,
) contractReverseDelta {
	var result contractReverseDelta
	for _, facet := range changedArtifactFacets(previous, next) {
		previousValue, previousPresent := previous.facet(facet)
		nextValue, nextPresent := next.facet(facet)
		prefix := commonPrefixLength(previousValue, nextValue)
		suffix := commonSuffixLength(
			previousValue[prefix:],
			nextValue[prefix:],
		)
		previousEnd := len(previousValue) - suffix
		result.facets[facet] = facetReverseDelta{
			previousPresent: previousPresent,
			prefix:          prefix,
			suffix:          suffix,
			previousMiddle: compressHistoryBytes(
				previousValue[prefix:previousEnd],
			),
		}
		if !nextPresent {
			result.facets[facet].prefix = 0
			result.facets[facet].suffix = 0
		}
		result.changed |= uint8(1) << facet
	}
	return result
}

func (d contractReverseDelta) restore(next Contract) Contract {
	previous := next
	for facet := api.ArtifactFacetCallableSignature; facet <= api.ArtifactFacetExportSurface; facet++ {
		if d.changed&(uint8(1)<<facet) == 0 {
			continue
		}
		change := d.facets[facet]
		nextValue, nextPresent := next.facet(facet)
		if !nextPresent {
			nextValue = nil
		}
		if change.prefix+change.suffix > len(nextValue) {
			panic("artifact contract history delta is corrupt")
		}
		if change.previousPresent {
			middle := change.previousMiddle.exact()
			value := make(
				[]byte,
				change.prefix+len(middle)+change.suffix,
			)
			copied := copy(value, nextValue[:change.prefix])
			copied += copy(value[copied:], middle)
			copy(
				value[copied:],
				nextValue[len(nextValue)-change.suffix:],
			)
			previous.facets[facet] = value
			previous.present |= uint8(1) << facet
		} else {
			previous.facets[facet] = nil
			previous.present &^= uint8(1) << facet
		}
	}
	return previous
}

func (h contractHistory) retainedPayloadBytes() int {
	total := 0
	for _, entry := range h.entries {
		for facet := api.ArtifactFacetCallableSignature; facet <= api.ArtifactFacetExportSurface; facet++ {
			total += len(entry.previous.facets[facet].previousMiddle.data)
		}
	}
	return total
}

func commonPrefixLength(left []byte, right []byte) int {
	limit := min(len(left), len(right))
	index := 0
	for index < limit && left[index] == right[index] {
		index++
	}
	return index
}

func commonSuffixLength(left []byte, right []byte) int {
	limit := min(len(left), len(right))
	index := 0
	for index < limit &&
		left[len(left)-index-1] == right[len(right)-index-1] {
		index++
	}
	return index
}

func fingerprintContract(contract Contract) contractFingerprint {
	first := uint64(14695981039346656037)
	second := uint64(7809847782465536322)
	mix := func(value byte) {
		first ^= uint64(value)
		first *= 1099511628211
		second ^= uint64(value)
		second *= 14029467366897019727
		second ^= second >> 29
	}
	mix(contract.present)
	for facet := api.ArtifactFacetCallableSignature; facet <= api.ArtifactFacetExportSurface; facet++ {
		value, present := contract.facet(facet)
		mix(byte(facet))
		if !present {
			mix(0)
			continue
		}
		mix(1)
		length := uint64(len(value))
		for shift := uint(0); shift < 64; shift += 8 {
			mix(byte(length >> shift))
		}
		for index := 0; index < len(value); index++ {
			mix(value[index])
		}
	}
	return contractFingerprint{first: first, second: second}
}

func compressHistoryBytes(value []byte) exactHistoryBytes {
	if len(value) == 0 {
		return exactHistoryBytes{}
	}
	if len(value) < 256 {
		return exactHistoryBytes{
			data:      bytes.Clone(value),
			rawLength: len(value),
		}
	}
	var output bytes.Buffer
	writer := contractHistoryWriterPool.Get().(*flate.Writer)
	writer.Reset(&output)
	_, writeErr := writer.Write(value)
	closeErr := writer.Close()
	writer.Reset(io.Discard)
	contractHistoryWriterPool.Put(writer)
	if writeErr != nil {
		panic(writeErr)
	}
	if closeErr != nil {
		panic(closeErr)
	}
	if output.Len() >= len(value) {
		return exactHistoryBytes{
			data:      bytes.Clone(value),
			rawLength: len(value),
		}
	}
	return exactHistoryBytes{
		data:       bytes.Clone(output.Bytes()),
		rawLength:  len(value),
		compressed: true,
	}
}

func (b exactHistoryBytes) exact() []byte {
	if b.rawLength == 0 {
		return nil
	}
	if !b.compressed {
		return bytes.Clone(b.data)
	}
	reader := flate.NewReader(bytes.NewReader(b.data))
	result := make([]byte, b.rawLength)
	read, err := io.ReadFull(reader, result)
	if err != nil || read != len(result) {
		_ = reader.Close()
		panic("artifact contract history compressed payload is corrupt")
	}
	var trailing [1]byte
	trailingRead, trailingErr := reader.Read(trailing[:])
	closeErr := reader.Close()
	if trailingRead != 0 || trailingErr != io.EOF || closeErr != nil {
		panic("artifact contract history compressed payload has trailing data")
	}
	return result
}
