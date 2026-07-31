export type ClockHandle = ReturnType<typeof setTimeout>;

export function wallMilliseconds(): number {
  return Date.now();
}

export function monotonicMilliseconds(): number {
  return performance.now();
}

export function schedule(delayMilliseconds: number, action: () => void): ClockHandle {
  return setTimeout(action, Math.max(0, delayMilliseconds));
}

export function cancelSchedule(handle: ClockHandle): void {
  clearTimeout(handle);
}
