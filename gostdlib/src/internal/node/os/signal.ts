export function nodeSignal(name: string): NodeJS.Signals | undefined {
  switch (name) {
    case "hangup":
      return "SIGHUP";
    case "interrupt":
      return "SIGINT";
    case "quit":
      return "SIGQUIT";
    case "illegal instruction":
      return "SIGILL";
    case "trace/breakpoint trap":
      return "SIGTRAP";
    case "aborted":
      return "SIGABRT";
    case "bus error":
      return "SIGBUS";
    case "floating point exception":
      return "SIGFPE";
    case "killed":
      return "SIGKILL";
    case "user defined signal 1":
      return "SIGUSR1";
    case "segmentation fault":
      return "SIGSEGV";
    case "user defined signal 2":
      return "SIGUSR2";
    case "broken pipe":
      return "SIGPIPE";
    case "alarm clock":
      return "SIGALRM";
    case "terminated":
      return "SIGTERM";
    case "stack fault":
      return "SIGSTKFLT";
    case "child exited":
      return "SIGCHLD";
    case "continued":
      return "SIGCONT";
    case "stopped (signal)":
      return "SIGSTOP";
    case "stopped":
      return "SIGTSTP";
    case "stopped (tty input)":
      return "SIGTTIN";
    case "stopped (tty output)":
      return "SIGTTOU";
    case "urgent I/O condition":
      return "SIGURG";
    case "CPU time limit exceeded":
      return "SIGXCPU";
    case "file size limit exceeded":
      return "SIGXFSZ";
    case "virtual timer expired":
      return "SIGVTALRM";
    case "profiling timer expired":
      return "SIGPROF";
    case "window changed":
      return "SIGWINCH";
    case "I/O possible":
      return "SIGIO";
    case "power failure":
      return "SIGPWR";
    case "bad system call":
      return "SIGSYS";
    default:
      return undefined;
  }
}
