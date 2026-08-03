import type { GoError } from "@gotots/runtime/interface-value.js";
import type { Awaitable, gostring, int64 } from "@gotots/runtime/scalars.js";
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
    process.kill(receiver.Pid, selected);
    return undefined;
  } catch {
    return nodeError("operation", "signal");
  }
}

export async function signalProcessAsync(
  receiver: ProcessValue | undefined,
  signal: { String(): Awaitable<gostring> } | undefined,
): Promise<GoError | undefined> {
  if (receiver === undefined || signal === undefined) {
    return nodeError("invalid", "signal");
  }
  const name = await signal.String();
  const selected = nodeSignal(name);
  if (selected === undefined) {
    return nodeError("invalid", "signal");
  }
  try {
    process.kill(receiver.Pid, selected);
    return undefined;
  } catch {
    return nodeError("operation", "signal");
  }
}

export function getProcessID(): int64 {
  return process.pid;
}

export function exitProcess(code: int64): never {
  process.exit(code);
}
