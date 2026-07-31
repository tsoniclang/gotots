export function signalName(number: number): string {
  switch (number) {
    case 1:
      return "hangup";
    case 2:
      return "interrupt";
    case 3:
      return "quit";
    case 4:
      return "illegal instruction";
    case 5:
      return "trace/breakpoint trap";
    case 6:
      return "aborted";
    case 7:
      return "bus error";
    case 8:
      return "floating point exception";
    case 9:
      return "killed";
    case 10:
      return "user defined signal 1";
    case 11:
      return "segmentation fault";
    case 12:
      return "user defined signal 2";
    case 13:
      return "broken pipe";
    case 14:
      return "alarm clock";
    case 15:
      return "terminated";
    case 16:
      return "stack fault";
    case 17:
      return "child exited";
    case 18:
      return "continued";
    case 19:
      return "stopped (signal)";
    case 20:
      return "stopped";
    case 21:
      return "stopped (tty input)";
    case 22:
      return "stopped (tty output)";
    case 23:
      return "urgent I/O condition";
    case 24:
      return "CPU time limit exceeded";
    case 25:
      return "file size limit exceeded";
    case 26:
      return "virtual timer expired";
    case 27:
      return "profiling timer expired";
    case 28:
      return "window changed";
    case 29:
      return "I/O possible";
    case 30:
      return "power failure";
    case 31:
      return "bad system call";
    default:
      return `signal ${number}`;
  }
}

export function nodeSignalName(number: number): NodeJS.Signals | undefined {
  switch (number) {
    case 1:
      return "SIGHUP";
    case 2:
      return "SIGINT";
    case 3:
      return "SIGQUIT";
    case 4:
      return "SIGILL";
    case 5:
      return "SIGTRAP";
    case 6:
      return "SIGABRT";
    case 7:
      return "SIGBUS";
    case 8:
      return "SIGFPE";
    case 9:
      return "SIGKILL";
    case 10:
      return "SIGUSR1";
    case 11:
      return "SIGSEGV";
    case 12:
      return "SIGUSR2";
    case 13:
      return "SIGPIPE";
    case 14:
      return "SIGALRM";
    case 15:
      return "SIGTERM";
    case 16:
      return "SIGSTKFLT";
    case 17:
      return "SIGCHLD";
    case 18:
      return "SIGCONT";
    case 19:
      return "SIGSTOP";
    case 20:
      return "SIGTSTP";
    case 21:
      return "SIGTTIN";
    case 22:
      return "SIGTTOU";
    case 23:
      return "SIGURG";
    case 24:
      return "SIGXCPU";
    case 25:
      return "SIGXFSZ";
    case 26:
      return "SIGVTALRM";
    case 27:
      return "SIGPROF";
    case 28:
      return "SIGWINCH";
    case 29:
      return "SIGIO";
    case 30:
      return "SIGPWR";
    case 31:
      return "SIGSYS";
    default:
      return undefined;
  }
}
