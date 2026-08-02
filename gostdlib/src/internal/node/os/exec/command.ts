import { spawnSync } from "node:child_process";
import type { SpawnSyncOptionsWithBufferEncoding } from "node:child_process";
import type { GoError } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, uint8 } from "@gotots/runtime/scalars.js";
import {
  state as ioState,
} from "../../../../io.js";
import type { Reader, Writer } from "../../../../io.js";
import type { SysProcAttr } from "../../../../syscall.js";
import { Duration } from "../../../../time.js";
import {
  File,
  Process,
  ProcessState,
} from "../../../../os.js";
import { ProviderError } from "../../../runtime/error.js";
import {
  byteSlice,
  sliceValues,
} from "../../../runtime/slice.js";
import { ProviderInterfaceValue } from "../../../portable/io/value.js";

const emptyOutput = RuntimeSlice.nil<uint8>();
const startedCommands = new WeakSet<CommandValue>();

export interface CommandValue {
  Path: gostring;
  Args: RuntimeSlice<gostring>;
  Env: RuntimeSlice<gostring>;
  Dir: gostring;
  Stdin: Reader | undefined;
  Stdout: Writer | undefined;
  Stderr: Writer | undefined;
  ExtraFiles: RuntimeSlice<File | undefined>;
  SysProcAttr: SysProcAttr | undefined;
  Process: Process | undefined;
  ProcessState: ProcessState | undefined;
  Err: GoError | undefined;
  Cancel: (() => Promise<GoError | undefined>) | undefined;
  WaitDelay: Duration;
}

export function commandOutput(
  receiver: CommandValue | undefined,
): [RuntimeSlice<uint8>, GoError | undefined] {
  if (receiver === undefined) {
    return [emptyOutput, new ProviderError("exec: nil Cmd")];
  }
  if (receiver.Err !== undefined) {
    return [emptyOutput, receiver.Err];
  }
  if (receiver.Stdout !== undefined) {
    return [emptyOutput, new ProviderError("exec: Stdout already set")];
  }
  const stdout = new CaptureWriter();
  receiver.Stdout = stdout;
  if (startedCommands.has(receiver)) {
    return [emptyOutput, new ProviderError("exec: already started")];
  }
  startedCommands.add(receiver);
  if (receiver.Cancel !== undefined) {
    return [
      emptyOutput,
      new ProviderError(
        "exec: command with a non-nil Cancel was not created with CommandContext",
      ),
    ];
  }
  const attributeError = validateAttributes(receiver.SysProcAttr);
  if (attributeError !== undefined) {
    return [emptyOutput, attributeError];
  }

  const options = spawnOptions(receiver);
  const [input, inputError] = readStandardInput(receiver.Stdin);
  if (inputError !== undefined) {
    return [emptyOutput, inputError];
  }
  if (input !== undefined) {
    options.input = input;
  }
  const arguments_ = sliceValues(receiver.Args).slice(1);
  const result = spawnSync(receiver.Path, arguments_, options);
  receiver.Process = result.pid === undefined ? undefined : new Process(result.pid);
  receiver.ProcessState = new ProcessState();

  const stderr = receiver.Stderr ?? new CaptureWriter();
  receiver.Stderr = stderr;
  if (result.stderr !== undefined && result.stderr !== null) {
    const writeError = writeStandardError(stderr, result.stderr);
    if (writeError !== undefined) {
      return [emptyOutput, writeError];
    }
  }
  if (result.error !== undefined) {
    return [emptyOutput, new ProviderError(result.error.message)];
  }

  if (result.stdout !== undefined && result.stdout !== null) {
    const [, writeError] = stdout.Write(byteSlice(result.stdout));
    if (writeError !== undefined) {
      return [emptyOutput, writeError];
    }
  }
  const output = stdout.Bytes();
  if (result.status !== 0) {
    if (result.signal !== null) {
      return [
        output,
        new ProviderError(`signal: ${result.signal}`),
      ];
    }
    return [
      output,
      new ProviderError(`exit status ${result.status ?? 0}`),
    ];
  }
  return [output, undefined];
}

function spawnOptions(command: CommandValue): SpawnSyncOptionsWithBufferEncoding {
  const options: SpawnSyncOptionsWithBufferEncoding = {
    encoding: "buffer",
    maxBuffer: 64 * 1024 * 1024,
  };
  if (command.Dir.length > 0) {
    options.cwd = command.Dir;
  }
  if (!command.Env.isNil()) {
    options.env = environment(sliceValues(command.Env));
  }
  if (command.SysProcAttr?.Credential !== undefined) {
    options.uid = command.SysProcAttr.Credential.Uid;
    options.gid = command.SysProcAttr.Credential.Gid;
  }
  if (!command.ExtraFiles.isNil()) {
    const stdio: Array<"ignore" | "pipe" | number> = [
      "pipe",
      "pipe",
      "pipe",
    ];
    for (const file of sliceValues(command.ExtraFiles)) {
      stdio.push(file === undefined ? "ignore" : File.Fd(file));
    }
    options.stdio = stdio;
  }
  return options;
}

function environment(entries: readonly string[]): NodeJS.ProcessEnv {
  const result: NodeJS.ProcessEnv = {};
  for (const entry of entries) {
    const separator = entry.indexOf("=");
    if (separator < 0) {
      continue;
    }
    result[entry.slice(0, separator)] = entry.slice(separator + 1);
  }
  return result;
}

function validateAttributes(
  attributes: SysProcAttr | undefined,
): GoError | undefined {
  if (attributes === undefined) {
    return undefined;
  }
  if (
    attributes.Chroot.length > 0
    || attributes.Ptrace
    || attributes.Setsid
    || attributes.Setpgid
    || attributes.Setctty
    || attributes.Noctty
    || attributes.Foreground
    || attributes.Pgid !== 0
    || attributes.Pdeathsig.value !== 0
    || attributes.Cloneflags !== 0
    || attributes.Unshareflags !== 0
    || !attributes.UidMappings.isNil()
    || !attributes.GidMappings.isNil()
    || attributes.GidMappingsEnableSetgroups
    || !attributes.AmbientCaps.isNil()
    || attributes.UseCgroupFD
    || attributes.CgroupFD !== 0
    || attributes.PidFD !== undefined
    || (
      attributes.Credential !== undefined
      && !attributes.Credential.Groups.isNil()
    )
  ) {
    return new ProviderError("exec: SysProcAttr is not supported by Node");
  }
  return undefined;
}

const captureWriterType = Object.freeze({ comparable: true });

class CaptureWriter extends ProviderInterfaceValue implements Writer {
  readonly #content: number[] = [];

  constructor() {
    super(captureWriterType);
  }

  Write(buffer: RuntimeSlice<uint8>): [number, GoError | undefined] {
    this.#content.push(...sliceValues(buffer));
    return [buffer.length, undefined];
  }

  Bytes(): RuntimeSlice<uint8> {
    return byteSlice(this.#content);
  }
}

function writeStandardError(
  writer: Writer | undefined,
  content: Uint8Array,
): GoError | undefined {
  if (writer !== undefined && content.length > 0) {
    const [, error] = writer.Write(byteSlice(content));
    return error;
  }
  return undefined;
}

function readStandardInput(
  reader: Reader | undefined,
): [Uint8Array | undefined, GoError | undefined] {
  if (reader === undefined) {
    return [undefined, undefined];
  }
  const chunks: number[] = [];
  let emptyReads = 0;
  while (true) {
    const buffer = RuntimeSlice.make<uint8>(32 * 1024, null, 0);
    const [count, error] = reader.Read(buffer);
    for (let index = 0; index < count; index += 1) {
      chunks.push(buffer.get(index));
    }
    if (error !== undefined) {
      return error === ioState.EOF
        ? [Uint8Array.from(chunks), undefined]
        : [undefined, error];
    }
    if (count === 0) {
      emptyReads += 1;
      if (emptyReads >= 100) {
        return [
          undefined,
          new ProviderError("exec: Stdin returned no data"),
        ];
      }
    } else {
      emptyReads = 0;
    }
  }
}
