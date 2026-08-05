import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring, int } from "@gotots/gostdlib/internal/scalars.js";
import type { Writer } from "../io.js";
import {
  beginCpuProfile,
  finishCpuProfile,
  knownProfile,
  ProfileNameKey,
  profileSnapshot,
} from "../internal/node/runtime/profile.js";
import { ProviderError } from "../internal/runtime/error.js";
import { byteSlice } from "../internal/runtime/slice.js";

export class Profile {
  readonly [ProfileNameKey]: gostring;

  constructor(name: gostring) {
    this[ProfileNameKey] = name;
  }

  static WriteTo(
    receiver: Profile | undefined,
    w: Writer | undefined,
    debug: int,
  ): GoError | undefined {
    void debug;
    if (receiver === undefined || w === undefined) {
      return new ProviderError("pprof: nil profile or writer");
    }
    return write(w, profileSnapshot(receiver[ProfileNameKey]));
  }
}

export function Lookup(name: gostring): Profile | undefined {
  return knownProfile(name) ? new Profile(name) : undefined;
}

export function StartCPUProfile(w: Writer | undefined): GoError | undefined {
  if (w === undefined) {
    return new ProviderError("pprof: nil writer");
  }
  if (!beginCpuProfile(async (content): Promise<void> => {
    write(w, content);
  })) {
    return new ProviderError("cpu profiling already in use");
  }
  return undefined;
}

export async function StopCPUProfile(): Promise<void> {
  await finishCpuProfile();
}

function write(writer: Writer, content: Uint8Array): GoError | undefined {
  const [count, failure] = writer.Write(byteSlice(content));
  if (failure !== undefined) {
    return failure;
  }
  return count === BigInt(content.length)
    ? undefined
    : new ProviderError("pprof: short write");
}
