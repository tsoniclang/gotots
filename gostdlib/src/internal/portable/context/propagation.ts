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
  subscribeCancel(parentDone, childDone, () => {
    const failure = parentFailure();
    if (failure === undefined) {
      GoPanic.raiseRuntime("context: internal error: missing cancel error");
    }
    cancel(failure, parentCause() ?? failure);
  });
}

type CancelSubscription = "none" | "parent" | "child" | "observed";

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
  let unobserveParent = (): void => undefined;
  let unobserveChild = (): void => undefined;
  const claim = (apply: () => void): void => {
    if (claimed) {
      return;
    }
    claimed = true;
    unobserveParent();
    unobserveChild();
    apply();
  };
  unobserveParent = parentDone.$observeClose((): void => claim(applyParent));
  unobserveChild = childDone.$observeClose((): void => claim(() => undefined));
  return "observed";
}
