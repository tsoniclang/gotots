import { Errno } from "@gotots/gostdlib/syscall.js";
import type { uintptr } from "@gotots/gostdlib/internal/scalars.js";

const ENOSYS = 38n;

export function Syscall(
  _trap: uintptr,
  _a1: uintptr,
  _a2: uintptr,
  _a3: uintptr,
): [uintptr, uintptr, Errno] {
  return [0n, 0n, new Errno(ENOSYS)];
}

export function Syscall6(
  _trap: uintptr,
  _a1: uintptr,
  _a2: uintptr,
  _a3: uintptr,
  _a4: uintptr,
  _a5: uintptr,
  _a6: uintptr,
): [uintptr, uintptr, Errno] {
  return [0n, 0n, new Errno(ENOSYS)];
}

export function RawSyscall(
  trap: uintptr,
  a1: uintptr,
  a2: uintptr,
  a3: uintptr,
): [uintptr, uintptr, Errno] {
  return Syscall(trap, a1, a2, a3);
}
