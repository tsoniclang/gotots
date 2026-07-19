# Packages, Initialization And Concurrency

## Package Model

Each selected Go package has canonical identity, declaration graph,
initialization plan, generated modules, external edges, and product ownership.
Go import cycles are rejected by the frontend.

Generated ESM imports follow semantic package dependencies. Internal module
partitioning cannot change package visibility, initialization order, or live
package-variable behavior.

## Initialization Order

The initialization plan preserves:

1. dependency-package initialization;
2. package variable dependency ordering;
3. source-determined initialization order from the Go frontend;
4. multiple-value initializer behavior;
5. blank-identifier side effects;
6. `init` function order; and
7. panic behavior.

Natural ESM evaluation is used when the generated module graph proves the same
order and exactly-once behavior. Explicit generated initialization functions
are introduced only where module partitioning, cycles within generated shards,
lazy entry points, or separately assembled inputs make ESM observably
insufficient.

The explicit form declares zero-valued package storage before executing any
initializer, emits one `initPackage$()` with deterministic
`uninitialized/running/done` state, calls dependency initializers first,
executes variable and `init` steps in the frontend order, and is invoked by
every declared product root. Blank imports retain their initializer edge.
Panic does not turn initialization into a retry. Natural ESM is eligible only
when a verifier proves identical zero-before-initializer behavior and no TDZ
can be observed.

Initialization state is never selected by package-name checks.

## Package Variables

Package variables are shared mutable storage. Cross-module readers observe
current values. Address-taking yields stable storage identity.

The simplest form is a live ESM binding. Generated getter/setter/cell forms are
used only where assignment from another module, address escape, or assembly
boundaries require them.

## Generated Module Partition

Generated packages split deterministically at declaration and dependency-SCC
boundaries. Splitting does not cut a function body merely to satisfy a visual
line target. Canonical declaration IDs determine ownership so unrelated source
movement does not reshuffle the complete output.

Hand-maintained GoToTS source, tests, and specification files remain within the
repository line limit and split by semantic responsibility.

## Concurrency Scope

Goroutines, channels, send, receive, close, range over channel, and select are
Go language semantics. Imported synchronization libraries remain external.

The selected corpus contains a small but real set of concurrency operations.
GoToTS analyzes them mechanically by call/effect region before choosing a
runtime strategy. Their presence alone does not justify a program-wide
scheduler.

## Channel Semantics

A channel semantically retains:

- element type and direction;
- nil or live state;
- capacity and buffered values;
- blocked senders and receivers;
- close state;
- zero value on closed receive;
- comma-ok receive result; and
- ordering and select readiness.

Nil send/receive blocks, close of nil or closed channel panics, send on closed
channel panics, and receive from a closed drained channel returns zero/false.
Copies occur at send and receive boundaries.

## Concurrency Lowering Candidates

Candidate strategies are:

1. direct synchronous TypeScript when analysis proves operations cannot block
   and no goroutine is created;
2. localized async or state-machine lowering whose complete call/effect region
   is proven equivalent;
3. a shared typed cooperative scheduler/channel runtime when repeated selected
   behavior proves it is the smallest exact design; and
4. unimplemented.

Accepted manual ownership and external runtime binding are ownership choices,
not representation candidates; they follow `externals-manual-extensions.md`.
Direct, localized, and scheduler strategies may be incomparable and therefore
declare exact capability predicates and deterministic cost ordering.

The planner reviews the complete may-block call closure. It cannot make one
function async while leaving a synchronous caller that must suspend.

Localized async lowering cannot change a Go `func(...) R` boundary into a
Promise-returning TypeScript boundary. Suspension is propagated through one
closed internal state-machine/CPS ABI across the complete may-block region.
Function values, interfaces, external callbacks, or exported synchronous
boundaries that cannot participate make the region unimplemented rather than
introducing a second async signature.

## Goroutines

`go call()` evaluates the function value and arguments in the launching
goroutine, then schedules the invocation. Panics belong to the launched
goroutine. Exit and blocked-operation behavior follow the selected concurrency
implementation.

An unrecovered panic in any generated goroutine follows the pinned Go process
termination behavior; it is not converted to an ignored Promise rejection.

A localized implementation must still preserve interaction with every channel,
defer, panic, package-global, and external effect in its closure.

## Select

Select evaluates channel operands and send values once in source order before
choosing a case. Exactly one ready communication proceeds. Default proceeds
only when no communication can. Nil channels disable their cases.

When several cases are ready, the implementation follows a documented policy
inside Go's permitted selection behavior. Tests check the required outcome set
unless the product deliberately pins a stronger deterministic policy.

## Cooperative Scheduler Threshold

A shared scheduler is accepted only with evidence that:

- multiple selected operation regions require the same scheduling semantics;
- direct/local/manual forms would duplicate or contradict behavior;
- the scheduler covers the complete may-block closure;
- channel, select, panic, defer, and package-exit behavior are oracle-proven;
- no-extension hot paths pay more than a static absence check; and
- allocation, latency, and throughput meet `performance.md`.

Until that evidence exists, affected classes may remain manual or
unimplemented. Coverage pressure alone is not justification.

This specification does not by itself accept a shared scheduler/channel ABI.
Until a checked-in decision supplies its static types, state transitions,
boundary ABI, differential model, and performance evidence, every class that
requires that candidate is unimplemented.

## Memory And Ordering

Generated execution must be one behavior permitted by the Go memory model.
Race freedom is a committed selected-product precondition proved by the
required race/corpus evidence, not inferred from one successful run. A region
with potentially racing shared-memory behavior outside the accepted execution
model is unimplemented. Synchronization through channels and accepted external
primitives establishes required happens-before relationships.

True parallel execution is not required merely because Go permits it. If the
selected product requires parallel throughput, that requirement receives a
separate measured implementation decision without weakening language behavior.

## Timers, Filesystem And Process Events

Timers, filesystem watching, process execution, mutexes, atomics, wait groups,
and operating-system events are external library contracts unless the Go
frontend identifies a language intrinsic. Their emulation may integrate with a
selected concurrency runtime through typed static contracts.

## Deadlock And Cancellation

If a shared scheduler is selected, it detects the applicable all-goroutines
blocked state and reports the accepted Go runtime failure behavior. Cancellation
exists only where selected source or external contracts define it; host promise
cancellation is not inferred.

## Concurrency Proof

Every concurrency class records:

- operation and may-block closure;
- selected lowering level;
- channel state transitions;
- evaluation and copy timing;
- permitted scheduling outcomes;
- panic/deadlock behavior;
- external interactions;
- deterministic test controls; and
- performance evidence.

Unproven closure edges make the class unimplemented.
