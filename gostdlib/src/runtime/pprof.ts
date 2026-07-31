import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring, int64 } from "@gotots/runtime/scalars.js";
import type { Writer } from "../io.js";
import {
  finishCpuSample,
  knownProfile,
  profileSnapshot,
  startCpuSample,
  type CpuSample,
} from "../internal/node/runtime/profile.js";
import { ProviderError } from "../internal/runtime/error.js";
import { byteSlice } from "../internal/runtime/slice.js";

export class Profile {
  constructor(private readonly name: gostring) {}

  static WriteTo(
    receiver: Profile | undefined,
    w: Writer | undefined,
    debug: int64,
  ): GoError | undefined {
    void debug;
    if (receiver === undefined || w === undefined) {
      return new ProviderError("pprof: nil profile or writer");
    }
    return write(w, profileSnapshot(receiver.name));
  }
}

let activeWriter: Writer | undefined;
let activeSample: CpuSample | undefined;

export function Lookup(name: gostring): Profile | undefined {
  return knownProfile(name) ? new Profile(name) : undefined;
}

export function StartCPUProfile(w: Writer | undefined): GoError | undefined {
  if (w === undefined) {
    return new ProviderError("pprof: nil writer");
  }
  if (activeWriter !== undefined) {
    return new ProviderError("cpu profiling already in use");
  }
  activeWriter = w;
  activeSample = startCpuSample();
  return undefined;
}

export function StopCPUProfile(): void {
  const writer = activeWriter;
  const sample = activeSample;
  activeWriter = undefined;
  activeSample = undefined;
  if (writer !== undefined && sample !== undefined) {
    write(writer, finishCpuSample(sample));
  }
}

function write(writer: Writer, content: Uint8Array): GoError | undefined {
  const [count, failure] = writer.Write(byteSlice(content));
  if (failure !== undefined) {
    return failure;
  }
  return count === content.length
    ? undefined
    : new ProviderError("pprof: short write");
}
