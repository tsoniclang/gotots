# Hand-Port Status

Authored Go-first with the current generated bodies hidden (contract
step 2): ALL 28 OF 28 (A1-A5, B1-B15, C1-C3, D1-D5). Measurements are
produced mechanically by `gotots-calibrate -measure` (joins the derived
manifest to this directory, strips reviewer headers, writes
`calibration/measurements.json` with named denominators).

Notes on the non-obvious rows:
- B10 (tracedTypeAdapter.Display) is authored as the MANUAL-REQUIRED
  EXEMPLAR: the defer/recover suppression maps to try/catch as a
  reviewed manual body. The verdict stays manual-required; no automatic
  threshold derives from it. Its generated ratio (0.27) is a typed
  placeholder, not a translation — do not read it as compression.
- C1 / C3 are the specialization planning outputs: C1 records the
  primitive-key binding-family decision (the JS Map representation owns
  insertion order and allocation, hand-port ratio 0.27 because the
  representation absorbs the Go bookkeeping); C3 records that ADR-0007
  forms 1-2 suffice (parametric port, 0.90). The remaining work for
  both is the per-instantiation call-site evidence join, which lands
  with the shape-gate evaluation.
- D1 (keyToMessage, 318 KB Go) is transform-assisted per contract:
  generated deterministically by `tools/d1_transform.py` from the exact
  manifest span (2,153 cases, fail-closed case-count check), ratio 1.05
  versus the baseline's 1.20. Never excerpt-only.

The high-byte tail is fully authored:
- D4 (NewChecker, 16.7 KB): 1.00x — the mechanical constructor ports
  byte-for-byte (Maps for Go maps, .bind for method values).
- D2 (verifyCompilerOptions, 22.7 KB): 1.02x — closure-heavy
  validation, memoized arrows and rest parameters.
- D5 (structuredTypeRelatedToWorker, 36.7 KB): 1.12x — the relation
  engine's structural core; tagless switches become if/else chains and
  the two-result closure returns a tuple.
- Corpus-wide: 423,161 hand-port bytes over 400,491 Go bytes = 1.057x
  including every exception-class and high-byte fixture.

The differential-execution harness (strict typecheck + Go-versus-TS
event oracle per fixture) deliberately FOLLOWS the independent review:
ports are shape excerpts (free sibling identifiers, method syntax), and
the harness's ambient-declaration scheme must be built against the
reviewed port shapes, not before them.

Thresholds derive ONLY from reviewed ports; pending fixtures cannot
contribute to any budget until authored and reviewed (contract:
Threshold Policy). Current ordinary median (1.04 over the 6 authored
ordinary-verdict fixtures A1-A5 and D1) is provisional pending that
review.
