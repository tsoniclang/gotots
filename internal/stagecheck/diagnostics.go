package stagecheck

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

const diagnosticSampleLimit = 20

// problemSet retains fixed-size deterministic diagnostics while accounting
// for every residual. The digest is an order-independent multiset digest:
// each record hash is added modulo 2^256, then bound with the total count.
type problemSet struct {
	count   uint64
	sum     [sha256.Size]byte
	samples []string
}

func newProblemSet() *problemSet { return &problemSet{} }

func (p *problemSet) add(record string) {
	if p == nil {
		return
	}
	p.count++
	member := sha256.Sum256(
		append([]byte("gotots-stage-residual/v1\x00"), record...),
	)
	carry := uint16(0)
	for index := len(p.sum) - 1; index >= 0; index-- {
		total := uint16(p.sum[index]) + uint16(member[index]) + carry
		p.sum[index] = byte(total)
		carry = total >> 8
	}
	index := sort.SearchStrings(p.samples, record)
	p.samples = append(p.samples, "")
	copy(p.samples[index+1:], p.samples[index:])
	p.samples[index] = record
	if len(p.samples) > diagnosticSampleLimit {
		p.samples = p.samples[:diagnosticSampleLimit]
	}
}

func (p *problemSet) addf(format string, args ...any) {
	p.add(fmt.Sprintf(format, args...))
}

func (p *problemSet) empty() bool { return p == nil || p.count == 0 }

func (p *problemSet) digest() string {
	hash := sha256.New()
	fmt.Fprintf(hash, "gotots-stage-problem-set/v1|count=%d|", p.count)
	hash.Write(p.sum[:])
	return hex.EncodeToString(hash.Sum(nil))
}

func (p *problemSet) summary(prefix string) string {
	return fmt.Sprintf(
		"%s (residuals=%d digest=%s sample=%v)",
		prefix, p.count, p.digest(), p.samples,
	)
}

func (p *problemSet) verificationError(
	stage string,
	prefix string,
) error {
	if p.empty() {
		return nil
	}
	return &VerificationError{
		Stage: stage, Reason: p.summary(prefix),
	}
}
