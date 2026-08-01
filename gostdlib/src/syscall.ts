import type { GoPointer } from "@gotots/runtime/pointer.js";
import { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  bool,
  gostring,
  int64,
  uint32,
  uint64,
} from "@gotots/runtime/scalars.js";
import { errnoMessage } from "./internal/node/syscall/errno.js";
import { signalName } from "./internal/node/syscall/signal.js";

const signalType = Object.freeze({});

class SignalInterfaceValue extends GoInterfaceValue {
  readonly $go$type: object = signalType;
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
    return this.value;
  }

  $go$format(
    verb: string,
    _flags: string,
    _precision: number | undefined,
  ): string {
    if (verb === "T") {
      return "syscall.Signal";
    }
    const name = signalName(this.value);
    return verb === "q" ? JSON.stringify(name) : name;
  }
}

export class Errno {
  constructor(public readonly value: uint64) {}

  Error(): gostring {
    return errnoMessage(this.value);
  }
}

export class Signal extends SignalInterfaceValue {
  constructor(value: int64) {
    super(value);
  }

  Signal(): void {}

  String(): gostring {
    return signalName(this.value);
  }
}

export const EINTR = new Errno(4);
export const ENOTDIR = new Errno(20);
export const EPERM = new Errno(1);
export const SIGINT = new Signal(2);
export const SIGTERM = new Signal(15);

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
    public ContainerID: int64 = 0,
    public HostID: int64 = 0,
    public Size: int64 = 0,
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
    public Ctty: int64 = 0,
    public Foreground: bool = false,
    public Pgid: int64 = 0,
    public Pdeathsig: Signal = new Signal(0),
    public Cloneflags: uint64 = 0,
    public Unshareflags: uint64 = 0,
    public UidMappings: RuntimeSlice<SysProcIDMap> = RuntimeSlice.nil<SysProcIDMap>(),
    public GidMappings: RuntimeSlice<SysProcIDMap> = RuntimeSlice.nil<SysProcIDMap>(),
    public GidMappingsEnableSetgroups: bool = false,
    public AmbientCaps: RuntimeSlice<uint64> = RuntimeSlice.nil<uint64>(),
    public UseCgroupFD: bool = false,
    public CgroupFD: int64 = 0,
    public PidFD: GoPointer<int64, int64> | undefined = undefined,
  ) {}
}
