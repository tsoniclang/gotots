import { Errno } from "@gotots/gostdlib/syscall.js";

const ENOSYS = 38;

export function Syscall(
  _trap: number,
  _a1: number,
  _a2: number,
  _a3: number,
): [number, number, Errno] {
  return [0, 0, new Errno(ENOSYS)];
}

export function Syscall6(
  _trap: number,
  _a1: number,
  _a2: number,
  _a3: number,
  _a4: number,
  _a5: number,
  _a6: number,
): [number, number, Errno] {
  return [0, 0, new Errno(ENOSYS)];
}

export function RawSyscall(
  trap: number,
  a1: number,
  a2: number,
  a3: number,
): [number, number, Errno] {
  return Syscall(trap, a1, a2, a3);
}
