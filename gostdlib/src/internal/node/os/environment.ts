import { GoString } from "@gotots/runtime/string-value.js";
import { fromHostString, toHostString } from "../../portable/utf8/codec.js";
import { homedir, tmpdir } from "node:os";
import { join } from "node:path";
import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring } from "@gotots/gostdlib/internal/scalars.js";
import { nodeError } from "./error.js";

export function executable(): [gostring, GoError | undefined] {
  return [fromHostString(process.execPath), undefined];
}

export function environment(name: gostring): gostring {
  return fromHostString(process.env[toHostString(name)] ?? "");
}

export function processArguments(): readonly gostring[] {
  return process.argv.slice(1).map(fromHostString);
}

export function workingDirectory(): [gostring, GoError | undefined] {
  try {
    return [fromHostString(process.cwd()), undefined];
  } catch {
    return [GoString.empty, nodeError("operation", "getwd")];
  }
}

export function temporaryDirectory(): gostring {
  return fromHostString(tmpdir());
}

export function userCacheDirectory(): [gostring, GoError | undefined] {
  const configured = process.env.XDG_CACHE_HOME;
  if (configured !== undefined && configured.length > 0) {
    return [fromHostString(configured), undefined];
  }
  const home = homedir();
  if (home.length === 0) {
    return [GoString.empty, nodeError("operation", "usercachedir")];
  }
  return [fromHostString(join(home, ".cache")), undefined];
}
