package source

import "fmt"

// SourceSpanHash is the SHA-256 of selected source bytes. It is acquisition
// evidence, distinct from semantic identity and generated/manual body hashes.
type SourceSpanHash [32]byte

func (h SourceSpanHash) IsZero() bool   { return h == SourceSpanHash{} }
func (h SourceSpanHash) String() string { return fmt.Sprintf("%x", h[:]) }
