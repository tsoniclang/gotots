# Machine Contracts And Diagnostics

## Schema Authority

Every durable report and product input uses a versioned checked-in schema with
canonical encoding. Schemas are part of the product contract and include:

- source/profile/toolchain attestation;
- selected census and canonical identities;
- declarations and signatures;
- semantic operations and classes;
- implementation support and ownership;
- effects and boundaries;
- value-flow regions and representation constraints;
- representation plans and necessity records;
- external obligations and bindings;
- manual body records;
- extension seams;
- test ledgers and results;
- generated artifacts and source maps;
- performance evidence; and
- decision registry.

Unknown schema versions and unknown required fields block. Optional fields have
one documented meaning and cannot serve as silent feature detection.

`schemas/manifest.json` is the exact schema inventory. Each listed contract has
one JSON Schema under `schemas/`, one canonical encoder/decoder, and positive,
negative, unknown-field, trailing-data, and mutation fixtures. Reviewed
semantic-class support decisions live in `contracts/support-classes.json` and
are joined to the generated operation census; an absent class is
unimplemented, never implicitly supported.

## Canonical Encoding

Canonical records use:

- UTF-8;
- normalized slash paths relative to an attested root;
- sorted object keys and identity-bearing arrays;
- exact decimal or hexadecimal encoding where numeric precision matters;
- lowercase validated cryptographic digests;
- explicit closed enums;
- no trailing data; and
- no timestamps or machine-local values inside content identity.

Hash identity includes schema version and semantic owner.

## Specification Manifest

`docs/spec/manifest.json` lists the complete normative file inventory and
reading order. Policy tests require:

- exactly the listed files plus the manifest;
- one H1 per Markdown file;
- valid internal links;
- no unlisted nested specification directory;
- no contradictory or historical design narration;
- repository file-size compliance; and
- agreement with README and accepted decision references.

The manifest is edited with the specification and has its own schema version.

## Identity-Bearing Ledgers

Every aggregate count derives from records with canonical IDs. Independent
ledgers cover:

- selected packages/files;
- declarations;
- implementation units;
- semantic operations;
- operation classes;
- test entities;
- external objects;
- external representation ABIs and adapters;
- manual bodies;
- extension seams;
- regions and representation plans;
- unimplemented units; and
- emitted artifacts.

Joins are exact one-to-one or explicitly typed one-to-many relations. Missing,
duplicate, orphaned, or multiply owned records block.

## Support Record

An implementation support record contains:

- canonical implementation ID;
- support state: generated, accepted-manual, or unimplemented;
- typed semantic-body and IR hashes;
- operation-class IDs;
- dependency closure;
- owner;
- emitted artifact IDs, if any;
- diagnostic IDs; and
- source/profile/tool revisions.

Body ownership state and automatic operation-class support are separate
ledgers. An unimplemented body record contains every unsupported operation-site
ID, not only the first one. Each site record contains semantic class, source
span, concise reason, missing accepted mechanism, and affected product roots.
It never points to a runnable owned-body stub.

## Representation Record

A representation record contains:

- region ID and all member value/storage IDs;
- semantic descriptors;
- complete operation set;
- accumulated product-lattice requirements and provenance;
- candidate capability predicates, legal conversions, and deterministic cost
  keys;
- selected candidate;
- explicit conversions;
- helper/specialization IDs;
- copy, equality, hash, bounds, nil, and panic behavior;
- necessity-record ID for custom output;
- oracle and performance evidence;
- dependency hashes; and
- verifier result.

The generator cannot set a bare “proven” flag. Evidence is structural and
independently checked.

## Necessity Record

Each custom mechanism record includes:

- mechanism kind and proof tier from the definition in
  `09-representation-output.md`;
- semantic class and mechanically generated site list;
- minimized Go counterexample;
- ordinary TypeScript candidate and observed mismatch;
- local rewrite, scalar expansion, specialization, and manual alternatives;
- selected smallest mechanism;
- semantic oracle IDs;
- mutation-test IDs;
- performance report IDs;
- invalidation dependencies;
- acceptance decision; and
- reopening conditions.

One mechanism may serve several classes only when the record proves their
semantic obligations identical.

Direct semantic IR nodes and stateless factored expressions do not require a
necessity record. Runtime-visible state, indirection, dispatch, scheduling, or
representation-changing adapters do. The verifier derives that requirement
from emitted AST/helper ownership rather than trusting a generator flag.

## Diagnostics

Diagnostics are stable structured records containing:

- code and schema version;
- severity and blocking scope;
- canonical source and semantic identities;
- exact source span;
- operation/type/representation context;
- concise human message;
- related evidence spans/records;
- affected artifact closure;
- remediation category; and
- input/tool revisions.

Human text can improve without changing code semantics. Tests primarily assert
codes and structured evidence.

## Diagnostic Families

Required families include:

- `GOTOTS_INPUT_*` — pin, toolchain, source, profile, and environment;
- `GOTOTS_SCOPE_*` — selected/outside dependency and ownership;
- `GOTOTS_CENSUS_*` — identity, disposition, and denominator;
- `GOTOTS_TYPE_*` — declaration and semantic-type evidence;
- `GOTOTS_IR_*` — unsupported or contradictory semantic operation;
- `GOTOTS_UNIMPLEMENTED_*` — recognized unsupported implementation class;
- `GOTOTS_REPRESENTATION_*` — constraints, fixed point, and necessity proof;
- `GOTOTS_EXTERNAL_*` — contract and binding;
- `GOTOTS_MANUAL_*` — ownership, drift, and effects;
- `GOTOTS_EXTENSION_*` — seam and assembly;
- `GOTOTS_OUTPUT_*` — path, AST, import, source map, and publication;
- `GOTOTS_TEST_*` — ledger, execution, count, and oracle;
- `GOTOTS_PERF_*` — baseline and regression; and
- `GOTOTS_UPGRADE_*` — revision delta and unclassified additions.

Catch-all internal errors are product failures and include a deterministic
failure bundle. They cannot be downgraded to unimplemented without identifying
the semantic class.

## Fail-Closed Boundaries

The following produce no affected product artifact:

- invalid or unattested input;
- unknown selected syntax/type/operation;
- selected dependency into outside scope;
- missing or conflicting identity;
- incomplete IR;
- nonconvergent or contradictory planning;
- custom mechanism without accepted necessity;
- unresolved reachable external;
- stale manual body;
- invalid extension seam;
- strict TypeScript failure;
- missing required test count; and
- required performance regression.

Independent packages may be emitted only in an explicitly incomplete analysis
bundle as defined by `01-input-census-publication.md`.

## Static Scanners

Repository policy scans complete maintained roots for:

- files over the line limit;
- forbidden generated dynamic/reflection mechanisms;
- CommonJS and triple-slash references;
- source-text semantic selection;
- raw generated-output patching;
- multiple ownership mechanisms;
- stale spec paths;
- unlisted specification files; and
- prohibited runtime fallback patterns.

Scanners support structural validation but do not substitute for typed semantic
analysis. False-positive suppressions require exact parser-backed evidence and
cannot become broad allowlists.

## No LLM Dependency

Source census, semantic classification, fixed-point propagation, code
generation, verification, diagnostics, and publication are deterministic
programs. They do not call an LLM or require per-site human judgment.

Human and LLM review may help design a semantic class or inspect evidence. The
accepted decision becomes checked-in machine rules, tests, and schemas before
it can affect generation.

## Failure Bundle

Every blocked or failed semantic gate can emit a deterministic bundle containing
the minimal relevant input identity, source span, typed IR fragment, graph
edges, constraints, selected or missing representation, generated AST fragment
if present, expected/actual oracle result, and all hashes.

Bundles use safe paths under `.temp/` and are not committed.
