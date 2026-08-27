import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring, int64 } from "@gotots/gostdlib/internal/scalars.js";
import {
  hostInteger,
  integerFromHost,
} from "../../host-integer.js";
import { nodeError } from "./error.js";
import { nodeSignal } from "./signal.js";

interface ProcessSignal {
  String(): gostring;
}

export interface ProcessValue {
  readonly Pid: int64;
}

export function signalProcess(
  receiver: ProcessValue | undefined,
  signal: ProcessSignal | undefined,
): GoError | undefined {
  if (receiver === undefined || signal === undefined) {
    return nodeError("invalid", "signal");
  }
  const name = signal.String();
  const selected = nodeSignal(name);
  if (selected === undefined) {
    return nodeError("invalid", "signal");
  }
  try {
    process.kill(hostInteger(receiver.Pid), selected);
    return undefined;
  } catch {
    return nodeError("operation", "signal");
  }
}

export function getProcessID(): int64 {
  return integerFromHost(process.pid);
}

export function exitProcess(code: int64): never {
  process.exit(hostInteger(code));
}
