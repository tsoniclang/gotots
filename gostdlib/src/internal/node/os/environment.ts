import { homedir, tmpdir } from "node:os";
import { join } from "node:path";
import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring } from "@gotots/gostdlib/internal/scalars.js";
import { nodeError } from "./error.js";

export function executable(): [gostring, GoError | undefined] {
  return [process.execPath, undefined];
}

export function environment(name: gostring): gostring {
  return process.env[name] ?? "";
}

export function processArguments(): readonly gostring[] {
  return process.argv.slice(1);
}

export function workingDirectory(): [gostring, GoError | undefined] {
  try {
    return [process.cwd(), undefined];
  } catch {
    return ["", nodeError("operation", "getwd")];
  }
}

export function temporaryDirectory(): gostring {
  return tmpdir();
}

export function userCacheDirectory(): [gostring, GoError | undefined] {
  const configured = process.env.XDG_CACHE_HOME;
  if (configured !== undefined && configured.length > 0) {
    return [configured, undefined];
  }
  const home = homedir();
  if (home.length === 0) {
    return ["", nodeError("operation", "usercachedir")];
  }
  return [join(home, ".cache"), undefined];
}
