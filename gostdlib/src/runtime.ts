import type {
  bool,
  gostring,
  int,
  uintptr,
} from "@gotots/gostdlib/internal/scalars.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import {
  caller,
  goArchitecture,
  goOperatingSystem,
  memorySnapshot,
} from "./internal/node/runtime/process.js";
import {
  MemStats,
  populateMemStats,
} from "./internal/portable/runtime/mem-stats.js";

export { MemStats };

export const GOOS: gostring = goOperatingSystem();
export const GOARCH: gostring = goArchitecture();

export function Caller(skip: int): [uintptr, gostring, int, bool] {
  return caller(skip);
}

export function GC(): void {}

export function GOMAXPROCS(n: int): int {
  void n;
  return 1n;
}

export function ReadMemStats(m: MemStats | undefined): void {
  if (m === undefined) {
    GoPanic.raiseRuntime("runtime.ReadMemStats called with nil receiver");
  }
  populateMemStats(m, memorySnapshot());
}
