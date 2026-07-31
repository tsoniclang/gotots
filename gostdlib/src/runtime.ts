import type {
  bool,
  gostring,
  int64,
  uint64,
} from "@gotots/runtime/scalars.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import {
  caller,
  goOperatingSystem,
  memorySnapshot,
} from "./internal/node/runtime/process.js";
import {
  MemStats,
  populateMemStats,
} from "./internal/portable/runtime/mem-stats.js";

export { MemStats };

export const GOOS: gostring = goOperatingSystem();

export function Caller(skip: int64): [uint64, gostring, int64, bool] {
  return caller(skip);
}

export function GC(): void {}

export function GOMAXPROCS(n: int64): int64 {
  void n;
  return 1;
}

export function ReadMemStats(m: MemStats | undefined): void {
  if (m === undefined) {
    GoPanic.raiseRuntime("runtime.ReadMemStats called with nil receiver");
  }
  populateMemStats(m, memorySnapshot());
}
