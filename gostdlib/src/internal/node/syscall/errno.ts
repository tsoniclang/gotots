import { getSystemErrorMessage } from "node:util";

export function errnoMessage(value: number): string {
  if (!Number.isSafeInteger(value) || value <= 0) {
    return `errno ${value}`;
  }
  try {
    return getSystemErrorMessage(-value);
  } catch {
    return `errno ${value}`;
  }
}
