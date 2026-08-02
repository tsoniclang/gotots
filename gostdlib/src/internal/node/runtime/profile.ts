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

export const ProfileNameKey: unique symbol = Symbol("gotots.pprof.profile-name");

export interface ProfileIdentity {
  readonly [ProfileNameKey]: string;
}

type CpuProfileWrite = (content: Uint8Array) => Promise<void>;

interface CpuProfileSession {
  readonly sample: CpuSample;
  readonly write: CpuProfileWrite;
}

let activeCpuProfile: CpuProfileSession | undefined;

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

export function beginCpuProfile(write: CpuProfileWrite): boolean {
  if (activeCpuProfile !== undefined) {
    return false;
  }
  activeCpuProfile = {
    sample: startCpuSample(),
    write,
  };
  return true;
}

export async function finishCpuProfile(): Promise<void> {
  const session = activeCpuProfile;
  activeCpuProfile = undefined;
  if (session !== undefined) {
    await session.write(finishCpuSample(session.sample));
  }
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
