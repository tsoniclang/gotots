import type { bool } from "@gotots/runtime/scalars.js";

import { Errno } from "../../syscall.js";
import { errnoMatchesSentinel } from "../portable/errors/sentinel.js";
import type { CanonicalError } from "./provider-io-contract.js";

export type { CanonicalError } from "./provider-io-contract.js";

export function SyscallErrnoIsCanonical(
  receiver: Errno,
  target: CanonicalError | undefined,
): bool {
  return errnoMatchesSentinel(receiver.value, target);
}
