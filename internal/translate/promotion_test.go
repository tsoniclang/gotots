// Embedded-pointer promotion differential proof (plan A4): a method or
// field promoted through an embedded POINTER delegates through Go's
// implicit dereference — sharing the pointee (writes through the promoted
// method are observed by the embedded target), and a nil embedded pointer
// panics at the dereference.
package translate_test

import "testing"

func TestOraclePromotionThroughEmbeddedPointer(t *testing.T) {
	runOracle(t, `package fixture

type opts struct{ N int }

func (o *opts) Bump() { o.N++ }
func (o *opts) Get() int { return o.N }

type parser struct {
	*opts
	tag int
}

func PromotedThroughPointer() int {
	base := opts{N: 5}
	p := parser{opts: &base, tag: 1}
	p.Bump()
	return base.N*100 + p.Get()*10 + p.N
}
`)
}

func TestOraclePromotedFieldSelection(t *testing.T) {
	// The tsoptions shape: selecting the embedded pointer itself and a
	// promoted field through it.
	runOracle(t, `package fixture

type inner struct{ V int }

type wrap struct {
	*inner
}

func PromotedFieldSelection() int {
	w := wrap{inner: &inner{V: 3}}
	w.V = 8
	q := w.inner
	return q.V*10 + w.V
}
`)
}

func TestOracleChanFieldDeclarationCompletes(t *testing.T) {
	// A struct with a channel field materializes (the field is the opaque
	// nil-only carrier); bodies that never touch the channel run exactly.
	runOracle(t, `package fixture

type worker struct {
	done chan struct{}
	n    int
}

func ChanFieldDeclares() int {
	w := worker{n: 4}
	w.n += 3
	return w.n
}
`)
}

func TestOracleFloatKeyedMap(t *testing.T) {
	// Go float-key semantics exactly: NaN-keyed inserts are fresh,
	// unretrievable, undeletable entries that still count and range;
	// +0 and -0 are one key.
	runOracle(t, `package fixture

func nanv() float64 {
	z := 0.0
	return z / z
}

func negzero() float64 {
	minus := -1.0
	zero := 0.0
	return zero * minus
}

func FloatKeyedMap() int {
	m := map[float64]int{}
	m[1.5] = 10
	m[nanv()] = 20
	m[nanv()] = 30
	m[negzero()] = 40
	m[0] = 50
	total := len(m) * 1000
	total += m[1.5]
	if _, ok := m[nanv()]; ok {
		total += 100000
	}
	delete(m, nanv())
	total += len(m) * 10
	sum := 0
	for _, v := range m {
		sum += v
	}
	return total + sum
}
`)
}

func TestOracleIfaceKeyedMap(t *testing.T) {
	// The FileLike/triviaPositionKey shape: an interface key whose members
	// are all pointer implementers — Go's (dynamic type, pointer) key
	// equality, exactly: same pointer hits, distinct pointer misses,
	// delete/len/range exact.
	runOracle(t, `package fixture

type keyed interface{ Tag() int }

type nodeA struct{ id int }

func (a *nodeA) Tag() int { return a.id }

type nodeB struct{ id int }

func (b *nodeB) Tag() int { return b.id }

func IfaceKeyedMap() int {
	a1 := &nodeA{id: 1}
	a2 := &nodeA{id: 1}
	b1 := &nodeB{id: 2}
	m := map[keyed]int{}
	m[a1] = 10
	m[b1] = 20
	total := 0
	if v, ok := m[a1]; ok {
		total += v
	}
	if _, ok := m[a2]; ok {
		total += 100000
	}
	m[a1] = 30
	total += len(m) * 1000
	delete(m, b1)
	total += len(m) * 100
	sum := 0
	for k, v := range m {
		sum += k.Tag() + v
	}
	return total + sum
}
`)
}
