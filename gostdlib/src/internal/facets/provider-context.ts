import type { GoReceiveChannel } from "@gotots/runtime/channel.js";
import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoMapHash } from "@gotots/runtime/map.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { bool } from "@gotots/runtime/scalars.js";
import { GoEmptyStruct } from "@gotots/runtime/struct.js";

import { ProviderChannel } from "../portable/concurrency/channel.js";
import { propagateCancel } from "../portable/context/propagation.js";
import { Duration } from "../portable/time/duration.js";
import { After } from "../portable/time/timer.js";
import { Now, Time } from "../portable/time/time.js";
import { goInterfaceEqual } from "../runtime/interface.js";

export type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
} from "./provider-io-contract.js";

export interface CanonicalContextSync<Failure extends GoInterfaceValue>
  extends GoInterfaceValue {
  Deadline(recovery?: GoRecovery): [Time, bool];
  Done(recovery?: GoRecovery): GoReceiveChannel<GoEmptyStruct> | undefined;
  Err(recovery?: GoRecovery): Failure | undefined;
  Value(
    key: GoInterfaceValue | undefined,
    recovery?: GoRecovery,
  ): GoInterfaceValue | undefined;
}

abstract class ContextValue<Failure extends GoInterfaceValue>
	implements CanonicalContextSync<Failure> {
	static readonly comparable = true;
  readonly $go$methods: ReadonlySet<object>;
  readonly $go$formatString = false;

  constructor(contract: readonly object[]) {
    this.$go$methods = new Set(contract);
  }

	abstract readonly $go$type: { readonly comparable: boolean };
  abstract Deadline(recovery?: GoRecovery): [Time, bool];
  abstract Done(
    recovery?: GoRecovery,
  ): GoReceiveChannel<GoEmptyStruct> | undefined;
  abstract Err(recovery?: GoRecovery): Failure | undefined;
  abstract Cause(): Failure | undefined;
  abstract Value(
    key: GoInterfaceValue | undefined,
    recovery?: GoRecovery,
  ): GoInterfaceValue | undefined;

  $go$implements(contract: readonly object[]): boolean {
    return contract.every((method) => this.$go$methods.has(method));
  }

  abstract $go$equal(other: GoInterfaceValue): boolean;
  abstract $go$hash(): number;

  $go$format(
    verb: string,
    _flags: string,
    _precision: number | undefined,
  ): string {
    if (verb === "T") {
      return "context.Context";
    }
    return GoPanic.raiseRuntime("context value formatting is unsupported");
  }
}

class ValueContext<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContextSync<Failure>,
> extends ContextValue<Failure> {
  readonly $go$type = ValueContext;
  readonly #parent: Parent;
  readonly #key: GoInterfaceValue;
  readonly #value: GoInterfaceValue | undefined;

  constructor(
    parent: Parent,
    key: GoInterfaceValue,
    value: GoInterfaceValue | undefined,
    contract: readonly object[],
  ) {
    super(contract);
    this.#parent = parent;
    this.#key = key;
    this.#value = value;
  }

  Deadline(recovery?: GoRecovery): [Time, bool] {
    return this.#parent.Deadline(recovery);
  }

  Done(recovery?: GoRecovery): GoReceiveChannel<GoEmptyStruct> | undefined {
    return this.#parent.Done(recovery);
  }

  Err(recovery?: GoRecovery): Failure | undefined {
    return this.#parent.Err(recovery);
  }

  Cause(): Failure | undefined {
    return contextCause(this.#parent);
  }

  Value(
    key: GoInterfaceValue | undefined,
    recovery?: GoRecovery,
  ): GoInterfaceValue | undefined {
    return goInterfaceEqual(this.#key, key)
      ? this.#value
      : this.#parent.Value(key, recovery);
  }

  $go$equal(other: GoInterfaceValue): boolean {
    return this === other;
  }

  $go$hash(): number {
    return GoMapHash.object(this);
  }
}

class CancelContext<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContextSync<Failure>,
> extends ContextValue<Failure> {
  readonly #done = new ProviderChannel<GoEmptyStruct>(
    () => GoEmptyStruct.$zero(),
    (value) => GoEmptyStruct.$copy(value),
    0,
  );
  #failure: Failure | undefined;
  #cause: Failure | undefined;

  constructor(
    private readonly parent: Parent,
    private readonly deadline: Time | undefined,
    contract: readonly object[],
  ) {
    super(contract);
    propagateCancel(
      parent.Done(),
      this.#done,
      () => parent.Err(),
      () => contextCause(parent),
      (failure, cause) => this.cancel(failure, cause),
    );
  }

  readonly $go$type = CancelContext;

  Deadline(): [Time, bool] {
    if (this.deadline !== undefined) {
      return [this.deadline, true];
    }
    return this.parent.Deadline();
  }

  Done(): GoReceiveChannel<GoEmptyStruct> {
    return this.#done;
  }

  Err(): Failure | undefined {
    return this.#failure;
  }

  Cause(): Failure | undefined {
    return this.#cause;
  }

  Value(
    key: GoInterfaceValue | undefined,
    recovery?: GoRecovery,
  ): GoInterfaceValue | undefined {
    return this.parent.Value(key, recovery);
  }

  cancel(failure: Failure, cause: Failure): void {
    if (this.#failure !== undefined) {
      return;
    }
    this.#failure = failure;
    this.#cause = cause;
    this.#done.close();
  }

  $go$equal(other: GoInterfaceValue): boolean {
    return this === other;
  }

  $go$hash(): number {
    return GoMapHash.object(this);
  }
}

export function ContextWithValueCanonicalSync<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContextSync<Failure>,
>(
  parent: Parent | undefined,
  key: GoInterfaceValue | undefined,
  value: GoInterfaceValue | undefined,
  contextContract: readonly object[],
): CanonicalContextSync<Failure> {
  if (parent === undefined) {
    GoPanic.raiseRuntime("cannot create context from nil parent");
  }
  if (key === undefined) {
    GoPanic.raiseRuntime("nil key");
  }
  return new ValueContext(parent, key, value, contextContract);
}

export function ContextWithCancelCanonicalSync<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContextSync<Failure>,
>(
  parent: Parent | undefined,
  canceled: Failure | undefined,
  contextContract: readonly object[],
): [CanonicalContextSync<Failure>, (_recovery?: GoRecovery) => Promise<void>] {
  const requiredCanceled = requireFailure(canceled);
  const child = new CancelContext<Failure, Parent>(
    requireParent(parent),
    undefined,
    contextContract,
  );
  return [
    child,
    async (): Promise<void> => child.cancel(requiredCanceled, requiredCanceled),
  ];
}

export function ContextWithCancelCauseCanonicalSync<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContextSync<Failure>,
>(
  parent: Parent | undefined,
  canceled: Failure | undefined,
  contextContract: readonly object[],
): [
  CanonicalContextSync<Failure>,
  (cause: Failure | undefined, _recovery?: GoRecovery) => Promise<void>,
] {
  const requiredCanceled = requireFailure(canceled);
  const child = new CancelContext<Failure, Parent>(
    requireParent(parent),
    undefined,
    contextContract,
  );
  return [
    child,
    async (cause: Failure | undefined): Promise<void> =>
      child.cancel(requiredCanceled, cause ?? requiredCanceled),
  ];
}

export function ContextWithTimeoutCanonicalSync<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContextSync<Failure>,
>(
  parent: Parent | undefined,
  timeout: Duration,
  canceled: Failure | undefined,
  deadlineExceeded: Failure | undefined,
  contextContract: readonly object[],
): [CanonicalContextSync<Failure>, (_recovery?: GoRecovery) => Promise<void>] {
  const actualParent = requireParent(parent);
  const requiredCanceled = requireFailure(canceled);
  const requiredDeadline = requireFailure(deadlineExceeded);
  const requestedDeadline = Now().Add(timeout);
  const [parentDeadline, parentHasDeadline] = actualParent.Deadline();
  const deadline = parentHasDeadline && parentDeadline.Before(requestedDeadline)
    ? parentDeadline
    : requestedDeadline;
  const child = new CancelContext<Failure, Parent>(
    actualParent,
    deadline,
    contextContract,
  );
  void After(deadline.Sub(Now())).receive().then(() =>
    child.cancel(requiredDeadline, requiredDeadline));
  return [
    child,
    async (): Promise<void> => child.cancel(requiredCanceled, requiredCanceled),
  ];
}

export function ContextAfterFuncCanonicalSync<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContextSync<Failure>,
>(
  parent: Parent | undefined,
  callback: (() => Promise<void>) | undefined,
): () => Promise<bool> {
  const done = requireParent(parent).Done();
  let stopped = false;
  let started = false;
  if (done !== undefined) {
    void done.receive().then(async () => {
      if (!stopped) {
        started = true;
        if (callback === undefined) {
          GoPanic.raiseRuntime("context.AfterFunc called with nil function");
        }
        await callback();
      }
    });
  }
  return async (): Promise<bool> => {
    if (started || stopped) {
      return false;
    }
    stopped = true;
    return true;
  };
}

export function ContextCauseCanonicalSync<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContextSync<Failure>,
>(parent: Parent | undefined): Failure | undefined {
  return contextCause(requireParent(parent));
}

function requireParent<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContextSync<Failure>,
>(parent: Parent | undefined): Parent {
  if (parent === undefined) {
    GoPanic.raiseRuntime("cannot create context from nil parent");
  }
  return parent;
}

function requireFailure<Failure extends GoInterfaceValue>(
  failure: Failure | undefined,
): Failure {
  if (failure === undefined) {
    GoPanic.raiseRuntime("context provider error is absent");
  }
  return failure;
}

function contextCause<Failure extends GoInterfaceValue>(
  source: CanonicalContextSync<Failure>,
): Failure | undefined {
  return source instanceof ContextValue ? source.Cause() : source.Err();
}
