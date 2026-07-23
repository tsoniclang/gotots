# Cleanroom Provenance & Admission Ledger

Evidence only. This ledger is subordinate to `docs/spec/` and records (a) the
retired-architecture revisions available for mining and (b) every component
admitted into the cleanroom under `docs/spec/delivery.md` §Cleanroom Admission
Rule. It is not a compatibility contract and grants no old code authority.

## Retired-architecture evidence revisions

The old production compiler was deleted atomically on branch
`feat/cleanroom-rebuild-20260721` (commit `25fa5e1`). Its behavior survives in
git history at these revisions for use as counterexamples, oracle cases, and
calibration fixtures — never as an imported foundation:

| Revision | Ref | Holds | Mining purpose |
|---|---|---|---|
| `aa2ed79e4181761b4eef98037603a57d679be55b` | `main` | Full old pipeline (~38K LOC: IR, emitter, translator, ABI, census), old 14-chapter spec, attestations | Differential oracle protocol, identity/collision cases, fingerprinting, strict-TS service protocol, regeneration/withholding scenarios, size/shape measurements |
| `b00aa43bf3e826484c823364b7fd4793e194270b` | `feat/tsts-total-completion-20260718` | Object-model native-class cutover WIP | embedding-is-composition, static-vs-virtual dispatch, canonical method identity, RTTI≠instanceof, 44KB-vtable counterexample |
| `af2a53517cb207597aa39fadd26d0435b837ae6f` | `checkpoint/object-model-static-dispatch-20260721` | Static-dispatch checkpoint WIP | static-ordinary-dispatch counterexample, self-family union collapse |

## Admitted components

Per delivery.md, a candidate returns only through the Cleanroom Admission Rule
with a recorded origin revision, retained behavior, rejected old behavior, new
owner, test evidence, and deletion search. No file or WIP commit is imported
wholesale.

| Component | Origin revision | New owner | Retained behavior | Rejected old behavior | Test evidence |
|---|---|---|---|---|---|
| _(none yet — cleanroom is built fresh from the specification)_ | | | | | |
