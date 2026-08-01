import type { GoReceiveChannel } from "@gotots/runtime/channel.js";
import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { GoInterfaceValue as InterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { bool } from "@gotots/runtime/scalars.js";
import { GoEmptyStruct } from "@gotots/runtime/struct.js";
import { ProviderError } from "../../runtime/error.js";
import { ProviderChannel } from "../concurrency/channel.js";
import { After } from "../time/timer.js";
import { Duration } from "../time/duration.js";
import { Now, Time } from "../time/time.js";

const contextMethodToken = {};
const canceled = new ProviderError("context canceled");
const deadlineExceeded = new ProviderError("context deadline exceeded");

export interface Context extends GoInterfaceValue {
  Deadline(): [Time, bool];
  Done(): GoReceiveChannel<GoEmptyStruct> | undefined;
  Err(): GoError | undefined;
  Value(key: GoInterfaceValue | undefined): GoInterfaceValue | undefined;
}

abstract class ContextValue extends InterfaceValue implements Context {
  readonly $go$type: object = ContextValue;
  readonly $go$methods: ReadonlySet<object> = new Set<object>([contextMethodToken]);
  readonly $go$formatString = false;

  abstract Deadline(): [Time, bool];
  abstract Done(): GoReceiveChannel<GoEmptyStruct> | undefined;
  abstract Err(): GoError | undefined;
  abstract Value(key: GoInterfaceValue | undefined): GoInterfaceValue | undefined;

  $go$implements(contract: readonly object[]): boolean {
    return contract.every((token) => this.$go$methods.has(token));
  }

  $go$equal(other: GoInterfaceValue): boolean {
    return this === other;
  }

  $go$hash(): number {
    return 0;
  }

  $go$format(verb: string): string {
    if (verb === "T") {
      return "context.Context";
    }
    return GoPanic.raiseRuntime("context value formatting is unsupported");
  }
}

class EmptyContext extends ContextValue {
  Deadline(): [Time, bool] {
    return [new Time(), false];
  }

  Done(): undefined {
    return undefined;
  }

  Err(): undefined {
    return undefined;
  }

  Value(): undefined {
    return undefined;
  }
}

class CancelContext extends ContextValue {
  readonly #done = new ProviderChannel<GoEmptyStruct>(
    () => GoEmptyStruct.$zero(),
    (value) => GoEmptyStruct.$copy(value),
    0,
  );
  #failure: GoError | undefined;

  constructor(
    readonly parent: Context,
    readonly deadline: Time | undefined,
  ) {
    super();
    const parentFailure = parent.Err();
    if (parentFailure !== undefined) {
      this.cancel(parentFailure);
      return;
    }
    const parentDone = parent.Done();
    if (parentDone !== undefined) {
      void parentDone.receive().then(() => this.cancel(parent.Err() ?? canceled));
    }
  }

  Deadline(): [Time, bool] {
    if (this.deadline !== undefined) {
      return [this.deadline, true];
    }
    return this.parent.Deadline();
  }

  Done(): GoReceiveChannel<GoEmptyStruct> {
    return this.#done;
  }

  Err(): GoError | undefined {
    return this.#failure;
  }

  Value(key: GoInterfaceValue | undefined): GoInterfaceValue | undefined {
    return this.parent.Value(key);
  }

  cancel(failure: GoError): void {
    if (this.#failure !== undefined) {
      return;
    }
    this.#failure = failure;
    this.#done.close();
  }
}

class ValueContext extends ContextValue {
  constructor(
    readonly parent: Context,
    readonly key: GoInterfaceValue,
    readonly value: GoInterfaceValue | undefined,
  ) {
    super();
  }

  Deadline(): [Time, bool] {
    return this.parent.Deadline();
  }

  Done(): GoReceiveChannel<GoEmptyStruct> | undefined {
    return this.parent.Done();
  }

  Err(): GoError | undefined {
    return this.parent.Err();
  }

  Value(key: GoInterfaceValue | undefined): GoInterfaceValue | undefined {
    return interfaceEqual(this.key, key) ? this.value : this.parent.Value(key);
  }
}

const background = new EmptyContext();
const todo = new EmptyContext();

type CancelFuncABI = ((_recovery?: GoRecovery) => Promise<void>) | undefined;
type CancelCauseFuncABI = ((
    cause: GoError | undefined,
    _recovery?: GoRecovery,
  ) => Promise<void>) | undefined;

export type CancelFunc<
  $Value extends CancelFuncABI = CancelFuncABI,
> = $Value;
export type CancelCauseFunc<
  $Value extends CancelCauseFuncABI = CancelCauseFuncABI,
> = $Value;

export function Background(): Context {
  return background;
}

export function TODO(): Context {
  return todo;
}

export function WithCancel(
  parent: Context | undefined,
): [Context, NonNullable<CancelFunc>] {
  const child = new CancelContext(requireParent(parent), undefined);
  return [child, async (): Promise<void> => child.cancel(canceled)];
}

export function WithCancelCause(
  parent: Context | undefined,
): [Context, NonNullable<CancelCauseFunc>] {
  const child = new CancelContext(requireParent(parent), undefined);
  return [
    child,
    async (cause: GoError | undefined): Promise<void> => child.cancel(cause ?? canceled),
  ];
}

export function WithTimeout(
  parent: Context | undefined,
  timeout: Duration,
): [Context, NonNullable<CancelFunc>] {
  const actualParent = requireParent(parent);
  const requestedDeadline = Now().Add(timeout);
  const [parentDeadline, parentHasDeadline] = actualParent.Deadline();
  const deadline = parentHasDeadline && parentDeadline.Before(requestedDeadline)
    ? parentDeadline
    : requestedDeadline;
  const child = new CancelContext(actualParent, deadline);
  void After(deadline.Sub(Now())).receive().then(() => child.cancel(deadlineExceeded));
  return [child, async (): Promise<void> => child.cancel(canceled)];
}

export function WithValue(
  parent: Context | undefined,
  key: GoInterfaceValue | undefined,
  val: GoInterfaceValue | undefined,
): Context {
  if (key === undefined) {
    GoPanic.raiseRuntime("nil key");
  }
  return new ValueContext(requireParent(parent), key, val);
}

export function AfterFunc(
  ctx: Context | undefined,
  f: (() => Promise<void>) | undefined,
): () => Promise<bool> {
  const done = requireParent(ctx).Done();
  let stopped = false;
  let started = false;
  if (done !== undefined) {
    void done.receive().then(async () => {
      if (!stopped) {
        started = true;
        if (f === undefined) {
          GoPanic.raiseRuntime("context.AfterFunc called with nil function");
        }
        await f();
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

export const state: { Canceled: GoError } = { Canceled: canceled };

function requireParent(parent: Context | undefined): Context {
  if (parent === undefined) {
    GoPanic.raiseRuntime("cannot create context from nil parent");
  }
  return parent;
}

function interfaceEqual(
  left: GoInterfaceValue | undefined,
  right: GoInterfaceValue | undefined,
): boolean {
  return left === right
    || (left !== undefined && right !== undefined && left.$go$equal(right));
}
