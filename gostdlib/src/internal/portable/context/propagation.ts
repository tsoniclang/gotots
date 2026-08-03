import type { GoReceiveChannel } from "@gotots/runtime/channel.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoEmptyStruct } from "@gotots/runtime/struct.js";
import type { Awaitable } from "@gotots/runtime/scalars.js";

export function propagateCancel<Failure>(
  parentDone: GoReceiveChannel<GoEmptyStruct> | undefined,
  childDone: GoReceiveChannel<GoEmptyStruct>,
  parentFailure: () => Awaitable<Failure | undefined>,
  parentCause: () => Awaitable<Failure | undefined>,
  cancel: (failure: Failure, cause: Failure) => void,
): void {
  if (parentDone === undefined) {
    return;
  }
  const applyParent = async (): Promise<void> => {
    const failure = await parentFailure();
    if (failure === undefined) {
      GoPanic.raiseRuntime("context: internal error: missing cancel error");
    }
    cancel(failure, (await parentCause()) ?? failure);
  };
  const parentCase = parentDone.$selectReceive(() => void applyParent());
  if (parentCase.ready()) {
    parentCase.commit();
    return;
  }
  const childCase = childDone.$selectReceive(() => undefined);
  if (childCase.ready()) {
    childCase.commit();
    return;
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
}
