import { Errno } from "@gotots/gostdlib/syscall.js";
const ENOSYS = 38;
export function Syscall(_trap, _a1, _a2, _a3) {
    return [0, 0, new Errno(ENOSYS)];
}
export function Syscall6(_trap, _a1, _a2, _a3, _a4, _a5, _a6) {
    return [0, 0, new Errno(ENOSYS)];
}
export function RawSyscall(trap, a1, a2, a3) {
    return Syscall(trap, a1, a2, a3);
}
