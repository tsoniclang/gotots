import type { ProviderPointer } from "./internal/runtime/pointer.js";
import {
  type GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  bool,
  gostring,
  int,
  int64,
  uint32,
  uint64,
  uintptr,
} from "@gotots/gostdlib/internal/scalars.js";
import { hostInteger } from "./internal/host-integer.js";
import { errnoMessage } from "./internal/node/syscall/errno.js";
import { signalName } from "./internal/node/syscall/signal.js";
import {
  errnoMatchesSentinel,
} from "./internal/portable/errors/sentinel.js";

const signalType = Object.freeze({ comparable: true });

class SignalInterfaceValue extends GoInterfaceValue {
  readonly $go$type = signalType;
  readonly $go$methods: ReadonlySet<object> = new Set<object>();
  readonly $go$formatString = true;

  constructor(readonly value: int64) {
    super();
  }

  $go$implements(contract: readonly object[]): boolean {
    return contract.every((token: object): boolean => this.$go$methods.has(token));
  }

  $go$equal(other: GoInterfaceValue): boolean {
    return other instanceof SignalInterfaceValue && other.value === this.value;
  }

  $go$hash(): number {
    return Number(BigInt.asUintN(32, this.value));
  }

  $go$format(
    verb: string,
    _flags: string,
    _precision: number | undefined,
  ): string {
    if (verb === "T") {
      return "syscall.Signal";
    }
    const name = signalName(hostInteger(this.value));
    return verb === "q" ? JSON.stringify(name) : name;
  }
}

export class Errno {
  constructor(public readonly value: uint64) {}

  Error(): gostring {
    return errnoMessage(hostInteger(this.value));
  }

  Is(target: GoError | undefined): bool {
    return errnoMatchesSentinel(this.value, target);
  }

  Temporary(): bool {
    return this.value === 4n || this.value === 24n || this.value === 23n
      || this.Timeout();
  }

  Timeout(): bool {
    return this.value === 11n || this.value === 110n;
  }
}

export class Signal extends SignalInterfaceValue {
  constructor(value: int64) {
    super(value);
  }

  Signal(): void {}

  String(): gostring {
    return signalName(hostInteger(this.value));
  }
}

export const EAGAIN = new Errno(11n);
export const EINTR = new Errno(4n);
export const EINVAL = new Errno(22n);
export const ENOENT = new Errno(2n);
export const ENOTDIR = new Errno(20n);
export const EPERM = new Errno(1n);
export const SIGINT = new Signal(2n);
export const SIGTERM = new Signal(15n);

export class Credential {
  constructor(
    public Uid: uint32 = 0,
    public Gid: uint32 = 0,
    public Groups: RuntimeSlice<uint32> = RuntimeSlice.nil<uint32>(),
    public NoSetGroups: bool = false,
  ) {}
}

export class SysProcIDMap {
  constructor(
    public ContainerID: int = 0n,
    public HostID: int = 0n,
    public Size: int = 0n,
  ) {}
}

export class SysProcAttr {
  constructor(
    public Chroot: gostring = "",
    public Credential: Credential | undefined = undefined,
    public Ptrace: bool = false,
    public Setsid: bool = false,
    public Setpgid: bool = false,
    public Setctty: bool = false,
    public Noctty: bool = false,
    public Ctty: int = 0n,
    public Foreground: bool = false,
    public Pgid: int = 0n,
    public Pdeathsig: Signal = new Signal(0n),
    public Cloneflags: uintptr = 0n,
    public Unshareflags: uintptr = 0n,
    public UidMappings: RuntimeSlice<SysProcIDMap> = RuntimeSlice.nil<SysProcIDMap>(),
    public GidMappings: RuntimeSlice<SysProcIDMap> = RuntimeSlice.nil<SysProcIDMap>(),
    public GidMappingsEnableSetgroups: bool = false,
    public AmbientCaps: RuntimeSlice<uintptr> = RuntimeSlice.nil<uintptr>(),
    public UseCgroupFD: bool = false,
    public CgroupFD: int = 0n,
    public PidFD: ProviderPointer<int> | undefined = undefined,
  ) {}
}
