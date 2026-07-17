package typeid

import (
	"go/types"
	"os"
	"testing"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
)

// TestCorpusIdentityAudit is the reviewer's acceptance criterion: over
// every declaration and method type in the owned corpus, the canonical
// identity must agree with go/types.Identical — zero non-identical
// collisions (same Canonical, not Identical) and zero identical splits
// (Identical, different Canonical). Gated by GOTOTS_AUDIT (loads the
// corpus, which is slow). typeutil.Hash buckets Identical-equal types.
func TestCorpusIdentityAudit(t *testing.T) {
	dir := os.Getenv("GOTOTS_AUDIT")
	if dir == "" {
		t.Skip("set GOTOTS_AUDIT=<corpus dir> to run the corpus identity audit")
	}
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports,
		Dir:  dir,
	}
	pkgs, err := packages.Load(cfg, "./internal/...")
	if err != nil {
		t.Fatal(err)
	}
	var typesList []types.Type
	methodOf := map[types.Type]*types.Func{}
	seenObj := map[types.Object]bool{}
	add := func(x types.Type) {
		if x != nil {
			typesList = append(typesList, x)
		}
	}
	for _, p := range pkgs {
		if p.Types == nil {
			continue
		}
		scope := p.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if seenObj[obj] {
				continue
			}
			seenObj[obj] = true
			add(obj.Type())
			if named, ok := obj.Type().(*types.Named); ok {
				for i := range named.NumMethods() {
					m := named.Method(i)
					typesList = append(typesList, m.Type())
					methodOf[m.Type()] = m
				}
			}
		}
	}
	t.Logf("audited types: %d", len(typesList))

	// Bucket by Identical (via typeutil.Hash + Identical refinement).
	hasher := typeutil.MakeHasher()
	byHash := map[uint32][]types.Type{}
	for _, x := range typesList {
		h := hasher.Hash(x)
		byHash[h] = append(byHash[h], x)
	}
	// Identical classes -> set of distinct Canonicals (splits detector).
	// Canonical groups -> set of non-Identical members (collision detector).
	canonOf := map[types.Type]string{}
	declKey := map[types.Type]string{} // "same declaration" oracle
	skipped := 0
	for _, x := range typesList {
		var cstr string
		var err error
		if m, ok := methodOf[x]; ok {
			var shape string
			shape, err = MethodCanonical(m)
			owner := ""
			if recv := m.Type().(*types.Signature).Recv(); recv != nil {
				owner = recv.Type().String()
			}
			cstr = owner + "|" + m.Name() + "|" + shape
			declKey[x] = owner + "|" + m.Name()
		} else {
			cstr, err = Canonical(x)
			declKey[x] = "" // non-methods use types.Identical below
		}
		if err != nil {
			skipped++
			continue
		}
		canonOf[x] = cstr
	}
	splits, collisions := 0, 0
	// Splits: within each Identical class, two members of the SAME kind
	// (both free types, or both methods of the same owner+name) must share
	// one identity. A method and a free function may share a signature yet
	// are DIFFERENT declarations (the method identity carries owner+name),
	// so cross-kind pairs are not splits.
	for _, bucket := range byHash {
		var classes [][]types.Type
		for _, x := range bucket {
			placed := false
			for i := range classes {
				if types.Identical(classes[i][0], x) {
					classes[i] = append(classes[i], x)
					placed = true
					break
				}
			}
			if !placed {
				classes = append(classes, []types.Type{x})
			}
		}
		for _, class := range classes {
			for i := 0; i < len(class); i++ {
				for j := i + 1; j < len(class); j++ {
					a, b := class[i], class[j]
					ca, oka := canonOf[a]
					cb, okb := canonOf[b]
					if !oka || !okb {
						continue
					}
					_, aM := methodOf[a]
					_, bM := methodOf[b]
					if aM != bM {
						continue // method vs free type: different declarations
					}
					if aM && declKey[a] != declKey[b] {
						continue // different methods (owner/name) sharing a shape
					}
					if ca != cb {
						splits++
						if splits <= 8 {
							t.Errorf("SPLIT\nA=%q\nB=%q", ca, cb)
						}
					}
				}
			}
		}
	}
	// Collisions: group by Canonical, all must be mutually Identical.
	byCanon := map[string][]types.Type{}
	for x, c := range canonOf {
		byCanon[c] = append(byCanon[c], x)
	}
	desc := func(x types.Type) string {
		if m, ok := methodOf[x]; ok {
			return m.String()
		}
		return x.String()
	}
	for _, group := range byCanon {
		for i := 1; i < len(group); i++ {
			var bad bool
			if _, isM := methodOf[group[0]]; isM {
				bad = declKey[group[i]] != declKey[group[0]] // different method decls, same identity
			} else {
				bad = !types.Identical(group[0], group[i])
			}
			if bad {
				collisions++
				if collisions <= 6 {
					t.Errorf("COLLISION %q:\n  A: %s\n  B: %s", canonOf[group[0]], desc(group[0]), desc(group[i]))
				}
				break
			}
		}
	}
	// The canonicalized denominator is reported separately from the
	// discovered input count. Every discovered type must canonicalize
	// (canonicalized + explicitly-excluded == discovered); there is no
	// explicit-exclusion list, so a single silent skip fails the audit —
	// an unresolved identity is never quietly dropped from the comparison.
	// (distinct counts the interned type objects: identical types share
	// one pointer, so distinct < discovered is ordinary type interning.)
	discovered := len(typesList)
	canonicalized := discovered - skipped
	distinct := len(canonOf)
	t.Logf("discovered=%d canonicalized=%d excluded=0 distinct=%d splits=%d collisions=%d skipped(non-canonical)=%d",
		discovered, canonicalized, distinct, splits, collisions, skipped)
	if skipped > 0 {
		t.Fatalf("identity audit FAILED: %d types silently skipped (not canonicalized); every discovered type must resolve or be an explicit disposition", skipped)
	}
	if canonicalized != discovered {
		t.Fatalf("identity audit FAILED: canonicalized %d != discovered %d", canonicalized, discovered)
	}
	if splits > 0 || collisions > 0 {
		t.Fatalf("identity audit FAILED: %d splits, %d collisions", splits, collisions)
	}
}
