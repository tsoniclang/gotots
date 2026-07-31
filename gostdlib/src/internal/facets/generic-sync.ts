import type { GoRecovery } from "@gotots/runtime/panic.js";
import { GoPanic } from "@gotots/runtime/panic.js";

export function SyncOnceValueCooperative<Value>(
  operation: (() => Promise<Value>) | undefined,
  _recovery?: GoRecovery,
): () => Promise<Value> {
  let result: Promise<Value> | undefined;
  return (): Promise<Value> => {
    result ??= operation === undefined
      ? Promise.reject(GoPanic.createRuntime("sync.OnceValue called with nil function"))
      : operation();
    return result;
  };
}
