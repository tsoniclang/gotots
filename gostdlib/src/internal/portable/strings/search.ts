import type { bool, gostring, int64 } from "@gotots/runtime/scalars.js";

export function Contains(text: gostring, substring: gostring): bool {
  return text.includes(substring);
}

export function HasPrefix(text: gostring, prefix: gostring): bool {
  return text.startsWith(prefix);
}

export function HasSuffix(text: gostring, suffix: gostring): bool {
  return text.endsWith(suffix);
}

export function Index(text: gostring, substring: gostring): int64 {
  return text.indexOf(substring);
}
