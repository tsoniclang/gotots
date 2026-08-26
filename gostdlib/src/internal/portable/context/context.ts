import type { GoReceiveChannel } from "@gotots/runtime/channel.js";
import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { GoInterfaceValue as InterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool } from "@gotots/gostdlib/internal/scalars.js";
import { GoEmptyStruct } from "@gotots/runtime/struct.js";
import { ProviderError } from "../../runtime/error.js";
import { goInterfaceEqual } from "../../runtime/interface.js";
import { ProviderChannel } from "../concurrency/channel.js";
import { propagateCancel } from "./propagation.js";
import { AfterFunc as scheduleAfter } from "../time/timer.js";
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

export abstract class ContextValue extends InterfaceValue implements Context {
	static readonly comparable = true;
	readonly $go$type = ContextValue;
  readonly $go$methods: ReadonlySet<object> = new Set<object>([contextMethodToken]);
  readonly $go$formatString = false;

  abstract Deadline(): [Time, bool];
  abstract Done(): GoReceiveChannel<GoEmptyStruct> | undefined;
  abstract Err(): GoError | undefined;
  abstract Cause(): GoError | undefined;
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
    if (verb === "v" || verb === "s") {
      return this.$contextName();
    }
    return GoPanic.raise(
      new ProviderError(`context: unsupported format verb %${verb}`),
    );
  }

  abstract $contextName(): string;
}

// parentContextName resolves the exact Go context chain spelling; foreign
// context implementations keep the Go interface fallback.
function parentContextName(parent: Context): string {
  return parent instanceof ContextValue
    ? parent.$contextName()
    : "context.Context";
}

class EmptyContext extends ContextValue {
  constructor(private readonly name: string) {
    super();
  }

  $contextName(): string {
    return this.name;
  }

  Deadline(): [Time, bool] {
    return [new Time(), false];
  }

  Done(): undefined {
    return undefined;
  }

  Err(): undefined {
    return undefined;
  }

  Cause(): undefined {
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
  #cause: GoError | undefined;

  $contextName(): string {
    if (this.deadline === undefined) {
      return `${parentContextName(this.parent)}.WithCancel`;
    }
    return GoPanic.raise(
      new ProviderError(
        "context: deadline context formatting requires time formatting",
      ),
    );
  }

  constructor(
    readonly parent: Context,
    readonly deadline: Time | undefined,
  ) {
    super();
    propagateCancel(
      parent.Done(),
      this.#done,
      () => parent.Err(),
      () => contextCause(parent),
      (failure, cause) => this.cancel(failure, cause),
    );
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

  Cause(): GoError | undefined {
    return this.#cause;
  }

  Value(key: GoInterfaceValue | undefined): GoInterfaceValue | undefined {
    return this.parent.Value(key);
  }

  cancel(failure: GoError, cause: GoError): void {
    if (this.#failure !== undefined) {
      return;
    }
    this.#failure = failure;
    this.#cause = cause;
    this.#done.close();
  }
}

class ValueContext extends ContextValue {
  $contextName(): string {
    return GoPanic.raise(
      new ProviderError(
        "context: value context formatting requires key and value formatting",
      ),
    );
  }

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

  Cause(): GoError | undefined {
    return contextCause(this.parent);
  }

  Value(key: GoInterfaceValue | undefined): GoInterfaceValue | undefined {
    return goInterfaceEqual(this.key, key) ? this.value : this.parent.Value(key);
  }
}

const background = new EmptyContext("context.Background");
const todo = new EmptyContext("context.TODO");

export type CancelFunc = (() => void) | undefined;
export type CancelCauseFunc = ((
  cause: GoError | undefined,
) => void) | undefined;

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
  return [child, (): void => child.cancel(canceled, canceled)];
}

export function WithCancelCause(
  parent: Context | undefined,
): [Context, NonNullable<CancelCauseFunc>] {
  const child = new CancelContext(requireParent(parent), undefined);
  return [
    child,
    (cause: GoError | undefined): void =>
      child.cancel(canceled, cause ?? canceled),
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
  scheduleAfter(
    deadline.Sub(Now()),
    (): void => child.cancel(deadlineExceeded, deadlineExceeded),
  );
  return [child, (): void => child.cancel(canceled, canceled)];
}

export function Cause(ctx: Context | undefined): GoError | undefined {
  return contextCause(requireParent(ctx));
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
  f: (() => void) | undefined,
): () => bool {
  const done = requireParent(ctx).Done();
  let stopped = false;
  let started = false;
  let unsubscribe = (): void => undefined;
  if (done !== undefined) {
    const run = (): void => {
      if (f === undefined) {
        GoPanic.raiseRuntime("context.AfterFunc called with nil function");
      }
      f();
    };
    const selected = done.$selectReceive(run);
    if (selected.ready()) {
      started = true;
      selected.commit();
    } else {
      unsubscribe = done.$observeClose((): void => {
        if (stopped || started) {
          return;
        }
        started = true;
        run();
      });
    }
  }
  return (): bool => {
    if (started || stopped) {
      return false;
    }
    stopped = true;
    unsubscribe();
    return true;
  };
}

export const state: { Canceled: GoError; DeadlineExceeded: GoError } = {
  Canceled: canceled,
  DeadlineExceeded: deadlineExceeded,
};

function requireParent(parent: Context | undefined): Context {
  if (parent === undefined) {
    GoPanic.raiseRuntime("cannot create context from nil parent");
  }
  return parent;
}

function contextCause(source: Context): GoError | undefined {
  return source instanceof ContextValue ? source.Cause() : source.Err();
}
