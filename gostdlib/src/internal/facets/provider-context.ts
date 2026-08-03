import type { GoReceiveChannel } from "@gotots/runtime/channel.js";
import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoMapHash } from "@gotots/runtime/map.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { Awaitable, bool } from "@gotots/runtime/scalars.js";
import { GoEmptyStruct } from "@gotots/runtime/struct.js";

import { ProviderChannel } from "../portable/concurrency/channel.js";
import { propagateCancelAwaitable } from "../portable/context/propagation.js";
import { Duration } from "../portable/time/duration.js";
import { After } from "../portable/time/timer.js";
import { Now, Time } from "../portable/time/time.js";
import { goInterfaceEqual } from "../runtime/interface.js";

export type { CanonicalError } from "./provider-io-contract.js";

export interface CanonicalContext<Failure extends GoInterfaceValue>
  extends GoInterfaceValue {
  Deadline(recovery?: GoRecovery): Awaitable<[Time, bool]>;
  Done(
    recovery?: GoRecovery,
  ): Awaitable<GoReceiveChannel<GoEmptyStruct> | undefined>;
  Err(recovery?: GoRecovery): Awaitable<Failure | undefined>;
  Value(
    key: GoInterfaceValue | undefined,
    recovery?: GoRecovery,
  ): Awaitable<GoInterfaceValue | undefined>;
}

abstract class ContextValue<Failure extends GoInterfaceValue>
	implements CanonicalContext<Failure> {
	static readonly comparable = true;
  readonly $go$methods: ReadonlySet<object>;
  readonly $go$formatString = false;

  constructor(contract: readonly object[]) {
    this.$go$methods = new Set(contract);
  }

	abstract readonly $go$type: { readonly comparable: boolean };
  abstract Deadline(recovery?: GoRecovery): Awaitable<[Time, bool]>;
  abstract Done(
    recovery?: GoRecovery,
  ): Awaitable<GoReceiveChannel<GoEmptyStruct> | undefined>;
  abstract Err(recovery?: GoRecovery): Awaitable<Failure | undefined>;
  abstract Cause(): Awaitable<Failure | undefined>;
  abstract Value(
    key: GoInterfaceValue | undefined,
    recovery?: GoRecovery,
  ): Awaitable<GoInterfaceValue | undefined>;

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
  Parent extends CanonicalContext<Failure>,
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

  Deadline(recovery?: GoRecovery): Awaitable<[Time, bool]> {
    return this.#parent.Deadline(recovery);
  }

  Done(
    recovery?: GoRecovery,
  ): Awaitable<GoReceiveChannel<GoEmptyStruct> | undefined> {
    return this.#parent.Done(recovery);
  }

  Err(recovery?: GoRecovery): Awaitable<Failure | undefined> {
    return this.#parent.Err(recovery);
  }

  Cause(): Awaitable<Failure | undefined> {
    return contextCause(this.#parent);
  }

  Value(
    key: GoInterfaceValue | undefined,
    recovery?: GoRecovery,
  ): Awaitable<GoInterfaceValue | undefined> {
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
  Parent extends CanonicalContext<Failure>,
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
  }

  async Initialize(): Promise<this> {
    await propagateCancelAwaitable(
      await this.parent.Done(),
      this.#done,
      () => this.parent.Err(),
      () => contextCause(this.parent),
      (failure, cause) => this.cancel(failure, cause),
    );
    return this;
  }

  readonly $go$type = CancelContext;

  Deadline(): Awaitable<[Time, bool]> {
    if (this.deadline !== undefined) {
      return [this.deadline, true];
    }
    return this.parent.Deadline();
  }

  Done(): Awaitable<GoReceiveChannel<GoEmptyStruct>> {
    return this.#done;
  }

  Err(): Awaitable<Failure | undefined> {
    return this.#failure;
  }

  Cause(): Awaitable<Failure | undefined> {
    return this.#cause;
  }

  Value(
    key: GoInterfaceValue | undefined,
    recovery?: GoRecovery,
  ): Awaitable<GoInterfaceValue | undefined> {
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

export function ContextWithValueCanonical<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContext<Failure>,
>(
  parent: Parent | undefined,
  key: GoInterfaceValue | undefined,
  value: GoInterfaceValue | undefined,
  contextContract: readonly object[],
): CanonicalContext<Failure> {
  if (parent === undefined) {
    GoPanic.raiseRuntime("cannot create context from nil parent");
  }
  if (key === undefined) {
    GoPanic.raiseRuntime("nil key");
  }
  return new ValueContext(parent, key, value, contextContract);
}

export async function ContextWithCancelCanonical<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContext<Failure>,
>(
  parent: Parent | undefined,
  canceled: Failure | undefined,
  contextContract: readonly object[],
): Promise<[
  CanonicalContext<Failure>,
  (_recovery?: GoRecovery) => Awaitable<void>,
]> {
  const requiredCanceled = requireFailure(canceled);
  const child = await new CancelContext<Failure, Parent>(
    requireParent(parent),
    undefined,
    contextContract,
  ).Initialize();
  return [
    child,
    async (): Promise<void> => child.cancel(requiredCanceled, requiredCanceled),
  ];
}

export async function ContextWithCancelCauseCanonical<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContext<Failure>,
>(
  parent: Parent | undefined,
  canceled: Failure | undefined,
  contextContract: readonly object[],
): Promise<[
  CanonicalContext<Failure>,
  (cause: Failure | undefined, _recovery?: GoRecovery) => Awaitable<void>,
]> {
  const requiredCanceled = requireFailure(canceled);
  const child = await new CancelContext<Failure, Parent>(
    requireParent(parent),
    undefined,
    contextContract,
  ).Initialize();
  return [
    child,
    async (cause: Failure | undefined): Promise<void> =>
      child.cancel(requiredCanceled, cause ?? requiredCanceled),
  ];
}

export async function ContextWithTimeoutCanonical<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContext<Failure>,
>(
  parent: Parent | undefined,
  timeout: Duration,
  canceled: Failure | undefined,
  deadlineExceeded: Failure | undefined,
  contextContract: readonly object[],
): Promise<[
  CanonicalContext<Failure>,
  (_recovery?: GoRecovery) => Awaitable<void>,
]> {
  const actualParent = requireParent(parent);
  const requiredCanceled = requireFailure(canceled);
  const requiredDeadline = requireFailure(deadlineExceeded);
  const requestedDeadline = Now().Add(timeout);
  const [parentDeadline, parentHasDeadline] = await actualParent.Deadline();
  const deadline = parentHasDeadline && parentDeadline.Before(requestedDeadline)
    ? parentDeadline
    : requestedDeadline;
  const child = await new CancelContext<Failure, Parent>(
    actualParent,
    deadline,
    contextContract,
  ).Initialize();
  void After(deadline.Sub(Now())).receive().then(() =>
    child.cancel(requiredDeadline, requiredDeadline));
  return [
    child,
    async (): Promise<void> => child.cancel(requiredCanceled, requiredCanceled),
  ];
}

export async function ContextAfterFuncCanonical<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContext<Failure>,
>(
  parent: Parent | undefined,
  callback: (() => Awaitable<void>) | undefined,
): Promise<() => Awaitable<bool>> {
  const done = await requireParent(parent).Done();
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

export async function ContextCauseCanonical<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContext<Failure>,
>(parent: Parent | undefined): Promise<Failure | undefined> {
  return await contextCause(requireParent(parent));
}

function requireParent<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContext<Failure>,
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

async function contextCause<Failure extends GoInterfaceValue>(
  source: CanonicalContext<Failure>,
): Promise<Failure | undefined> {
  return await (source instanceof ContextValue ? source.Cause() : source.Err());
}
