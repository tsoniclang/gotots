// Package verify holds independent architecture and product gates. It may
// inspect every layer of the tree; production layers must never import it.
// Its checks derive their observations from source and final artifacts, not
// from the producers they verify.
package verify
