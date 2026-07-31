import { cpuSeconds, memorySnapshot, stackBytes } from "./process.js";
import {
  cpuProfile,
  emptyProfile,
  memoryProfile,
  stackProfile,
} from "./profile/pprof.js";

export interface CpuSample {
  readonly startedAt: number;
  readonly cpuAtStart: number;
}

export function startCpuSample(): CpuSample {
  return {
    startedAt: Date.now(),
    cpuAtStart: cpuSeconds(),
  };
}

export function finishCpuSample(sample: CpuSample): Uint8Array {
  const stoppedAt = Date.now();
  return cpuProfile(
    sample.startedAt,
    stoppedAt,
    Math.max(0, cpuSeconds() - sample.cpuAtStart),
  );
}

export function profileSnapshot(name: string): Uint8Array {
  switch (name) {
    case "allocs":
    case "heap":
      return memoryProfile(name, memorySnapshot());
    case "goroutine":
      return stackProfile(stackBytes());
    case "block":
    case "mutex":
    case "threadcreate":
      return emptyProfile(name);
    default:
      return new Uint8Array();
  }
}

export function knownProfile(name: string): boolean {
  switch (name) {
    case "allocs":
    case "block":
    case "goroutine":
    case "heap":
    case "mutex":
    case "threadcreate":
      return true;
    default:
      return false;
  }
}
