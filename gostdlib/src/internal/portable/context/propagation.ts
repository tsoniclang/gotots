import type { GoReceiveChannel } from "@gotots/runtime/channel.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoEmptyStruct } from "@gotots/runtime/struct.js";

export function propagateCancel<Failure>(
  parentDone: GoReceiveChannel<GoEmptyStruct> | undefined,
  childDone: GoReceiveChannel<GoEmptyStruct>,
  parentFailure: () => Failure | undefined,
  parentCause: () => Failure | undefined,
  cancel: (failure: Failure, cause: Failure) => void,
): void {
  if (parentDone === undefined) {
    return;
  }
  const applyParent = (): void => {
    const failure = parentFailure();
    if (failure === undefined) {
      GoPanic.raiseRuntime("context: internal error: missing cancel error");
    }
    cancel(failure, parentCause() ?? failure);
  };
  const parentCase = parentDone.$selectReceive(() => applyParent());
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
