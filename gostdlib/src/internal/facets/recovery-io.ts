import type { GoError } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int, uint8 } from "@gotots/gostdlib/internal/scalars.js";

import { Reader as BufioReader, Writer as BufioWriter } from "../../bufio.js";
import { Buffer as BytesBuffer } from "../../bytes.js";
import { Reader as GzipReader } from "../../compress/gzip.js";
import { PathError } from "../../io/fs.js";
import { File as OsFile } from "../../os.js";
import { Builder as StringsBuilder, Reader as StringsReader } from "../../strings.js";

export function BufioReaderRead(
  receiver: BufioReader | undefined,
  destination: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): [int, GoError | undefined] {
  return BufioReader.Read(receiver, destination);
}

export function BufioWriterWrite(
  receiver: BufioWriter | undefined,
  source: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): [int, GoError | undefined] {
  return BufioWriter.Write(receiver, source);
}

export function BytesBufferRead(
  receiver: BytesBuffer | undefined,
  destination: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): [int, GoError | undefined] {
  return BytesBuffer.Read(receiver, destination);
}

export function BytesBufferWrite(
  receiver: BytesBuffer | undefined,
  source: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): [int, GoError | undefined] {
  return BytesBuffer.Write(receiver, source);
}

export function GzipReaderClose(
  receiver: GzipReader | undefined,
  _recovery?: GoRecovery,
): GoError | undefined {
  return GzipReader.Close(receiver);
}

export function GzipReaderRead(
  receiver: GzipReader | undefined,
  destination: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): [int, GoError | undefined] {
  return GzipReader.Read(receiver, destination);
}

export function IoFsPathErrorError(
  receiver: PathError | undefined,
  _recovery?: GoRecovery,
): string {
  return PathError.Error(receiver);
}

export function OsFileClose(
  receiver: OsFile | undefined,
  _recovery?: GoRecovery,
): GoError | undefined {
  return OsFile.Close(receiver);
}

export function OsFileRead(
  receiver: OsFile | undefined,
  destination: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): [int, GoError | undefined] {
  return OsFile.Read(receiver, destination);
}

export function OsFileWrite(
  receiver: OsFile | undefined,
  source: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): [int, GoError | undefined] {
  return OsFile.Write(receiver, source);
}

export function StringsBuilderWrite(
  receiver: StringsBuilder | undefined,
  source: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): [int, GoError | undefined] {
  return StringsBuilder.Write(receiver, source);
}

export function StringsReaderRead(
  receiver: StringsReader | undefined,
  destination: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): [int, GoError | undefined] {
  return StringsReader.Read(receiver, destination);
}
