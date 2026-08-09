import { Cmd } from "../../os/exec.js";

export class OsExecCmdOperations {
  static $copy(source: Cmd): Cmd {
    const target = new Cmd();
    OsExecCmdOperations.$assign(target, source);
    return target;
  }

  static $assign(target: Cmd, source: Cmd): void {
    if (target === source) {
      return;
    }
    target.Path = source.Path;
    target.Args = source.Args;
    target.Env = source.Env;
    target.Dir = source.Dir;
    target.Stdin = source.Stdin;
    target.Stdout = source.Stdout;
    target.Stderr = source.Stderr;
    target.ExtraFiles = source.ExtraFiles;
    target.SysProcAttr = source.SysProcAttr;
    target.Process = source.Process;
    target.ProcessState = source.ProcessState;
    target.Err = source.Err;
    target.Cancel = source.Cancel;
    target.WaitDelay = source.WaitDelay;
  }
}
