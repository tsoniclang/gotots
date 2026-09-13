import { GoString } from "@gotots/runtime/string-value.js";
import { Buffer } from "node:buffer";
import { realpathSync } from "node:fs";

import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring } from "@gotots/gostdlib/internal/scalars.js";

import { providerError } from "../../runtime/error.js";
import {
  fromHostBytes,
  fromHostString,
  toHostBytes,
} from "../../portable/utf8/codec.js";
import {
  Clean,
  IsAbs,
  joinValues,
} from "../../portable/filepath/lexical.js";

export function Abs(path: gostring): [gostring, GoError | undefined] {
  try {
    return [IsAbs(path) ? Clean(path) : joinValues([fromHostString(process.cwd()), path]), undefined];
  } catch {
    return [GoString.empty, providerError(new Error("cannot determine absolute path"))];
  }
}

export function EvalSymlinks(path: gostring): [gostring, GoError | undefined] {
  try {
    const resolved = realpathSync.native(goPathBuffer(path), { encoding: "buffer" });
    const resolvedPath = bufferPath(resolved);
    if (IsAbs(path)) {
      return [Clean(resolvedPath), undefined];
    }
    const workingDirectory = fromHostString(process.cwd());
    if (resolvedPath.text() === workingDirectory.text()) {
      return [GoString.fromText("."), undefined];
    }
    const prefix = `${Clean(workingDirectory).text()}/`;
    return [
      resolvedPath.text().startsWith(prefix) ? Clean(resolvedPath.slice(prefix.length)) : Clean(resolvedPath),
      undefined,
    ];
  } catch {
    return [GoString.empty, providerError(new Error("cannot evaluate symbolic links"))];
  }
}

function goPathBuffer(path: gostring): Buffer {
  return Buffer.from(toHostBytes(path));
}

function bufferPath(path: Buffer): gostring {
  return fromHostBytes(path);
}
