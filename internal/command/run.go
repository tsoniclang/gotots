package command

import (
	"context"
	"io"

	"github.com/tsoniclang/gotots/internal/config"
)

func Run(
	ctx context.Context,
	workingDirectory string,
	arguments []string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	if ctx == nil || stdout == nil || stderr == nil {
		return commandError("run", "context or output stream is absent")
	}
	if len(arguments) != 0 && arguments[0] == compileWorkerCommand {
		return runCompileWorker(ctx, arguments[1:])
	}
	invocation, err := ParseArguments(workingDirectory, arguments)
	if err != nil {
		return err
	}
	project, err := config.Load(config.Request{
		ConfigPath: invocation.ConfigPath(),
		Overrides:  invocation.Overrides(),
	})
	if err != nil {
		return err
	}
	if invocation.PrintResolvedConfig() {
		payload, err := project.CanonicalJSON()
		if err != nil {
			return err
		}
		if _, err := stdout.Write(payload); err != nil {
			return commandError("print resolved config", err.Error())
		}
		return nil
	}
	report, err := Build(ctx, project)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(stdout, report.Summary()+"\n"); err != nil {
		return commandError("print build report", err.Error())
	}
	return nil
}
