import type { GoReceiveChannel } from "@gotots/runtime/channel.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoEmptyStruct } from "@gotots/runtime/struct.js";
import type { Awaitable } from "@gotots/gostdlib/internal/scalars.js";

export function propagateCancel<Failure>(
  parentDone: GoReceiveChannel<GoEmptyStruct> | undefined,
  childDone: GoReceiveChannel<GoEmptyStruct>,
  parentFailure: () => Failure | undefined,
  parentCause: () => Failure | undefined,
  cancel: (failure: Failure, cause: Failure) => void,
): void {
  subscribeCancel(parentDone, childDone, () => {
    const failure = parentFailure();
    if (failure === undefined) {
      GoPanic.raiseRuntime("context: internal error: missing cancel error");
    }
    cancel(failure, parentCause() ?? failure);
  });
}

export async function propagateCancelAwaitable<Failure>(
  parentDone: GoReceiveChannel<GoEmptyStruct> | undefined,
  childDone: GoReceiveChannel<GoEmptyStruct>,
  parentFailure: () => Awaitable<Failure | undefined>,
  parentCause: () => Awaitable<Failure | undefined>,
  cancel: (failure: Failure, cause: Failure) => void,
): Promise<void> {
  let application: Promise<void> | undefined;
  const disposition = subscribeCancel(parentDone, childDone, () => {
    application = applyAwaitableParent(parentFailure, parentCause, cancel);
  });
  if (disposition === "parent") {
    if (application === undefined) {
      GoPanic.raiseRuntime("context: internal error: parent cancellation was not applied");
    }
    await application;
  }
}

async function applyAwaitableParent<Failure>(
  parentFailure: () => Awaitable<Failure | undefined>,
  parentCause: () => Awaitable<Failure | undefined>,
  cancel: (failure: Failure, cause: Failure) => void,
): Promise<void> {
  const failure = await parentFailure();
  if (failure === undefined) {
    GoPanic.raiseRuntime("context: internal error: missing cancel error");
  }
  cancel(failure, (await parentCause()) ?? failure);
}

type CancelSubscription = "none" | "parent" | "child" | "subscribed";

function subscribeCancel(
  parentDone: GoReceiveChannel<GoEmptyStruct> | undefined,
  childDone: GoReceiveChannel<GoEmptyStruct>,
  applyParent: () => void,
): CancelSubscription {
  if (parentDone === undefined) {
    return "none";
  }
  const parentCase = parentDone.$selectReceive(applyParent);
  if (parentCase.ready()) {
    parentCase.commit();
    return "parent";
  }
  const childCase = childDone.$selectReceive(() => undefined);
  if (childCase.ready()) {
    childCase.commit();
    return "child";
  }
  let claimed = false;
  let unsubscribeParent = (): void => undefined;
  let unsubscribeChild = (): void => undefined;
  const claim = (): boolean => {
    if (claimed) {
      return false;
    }
    claimed = true;
    unsubscribeParent();
    unsubscribeChild();
    return true;
  };
  unsubscribeParent = parentCase.subscribe(claim);
  unsubscribeChild = childCase.subscribe(claim);
  return "subscribed";
}
