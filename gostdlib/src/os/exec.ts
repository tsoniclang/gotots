import type { GoError } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { Awaitable, gostring, uint8 } from "@gotots/gostdlib/internal/scalars.js";
import type { Reader, Writer } from "../io.js";
import type { SysProcAttr } from "../syscall.js";
import { Duration } from "../time.js";
import {
  File,
  Process,
  ProcessState,
} from "../os.js";
import { commandOutput } from "../internal/node/os/exec/command.js";
import {
  sliceValues,
  stringSlice,
} from "../internal/runtime/slice.js";

export class Cmd {
  Path: gostring = "";
  Args: RuntimeSlice<gostring> = RuntimeSlice.nil<gostring>();
  Env: RuntimeSlice<gostring> = RuntimeSlice.nil<gostring>();
  Dir: gostring = "";
  Stdin: Reader | undefined = undefined;
  Stdout: Writer | undefined = undefined;
  Stderr: Writer | undefined = undefined;
  ExtraFiles: RuntimeSlice<File | undefined> = RuntimeSlice.nil<File | undefined>();
  SysProcAttr: SysProcAttr | undefined = undefined;
  Process: Process | undefined = undefined;
  ProcessState: ProcessState | undefined = undefined;
  Err: GoError | undefined = undefined;
  Cancel: (() => Awaitable<GoError | undefined>) | undefined = undefined;
  WaitDelay: Duration = new Duration(0n);

  static Output(
    receiver: Cmd | undefined,
  ): [RuntimeSlice<uint8>, GoError | undefined] {
    return commandOutput(receiver);
  }
}

export function Command(
  name: gostring,
  arg: RuntimeSlice<gostring>,
): Cmd | undefined {
  const command = new Cmd();
  command.Path = name;
  command.Args = stringSlice([name, ...sliceValues(arg)]);
  return command;
}
