package hooks

import (
	"context"
	"fmt"
	"os"
	"time"
	"os/exec"

	"github.com/zrepl/zrepl/logger"
)

type contextKey int

const (
	contextKeyLog contextKey = 0
)

type Logger = logger.Logger

func getLogger(ctx context.Context) Logger {
	if log, ok := ctx.Value(contextKeyLog).(Logger); ok {
		return log
	}
	return logger.NewNullLogger()
}

func RunHookCommand(ctx context.Context, command string, env map[string]string, timeout time.Duration) error {
	var log = getLogger(ctx).
	    WithField("subsystem", "hooks")

	// Run hook executable and wait for timeout
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	commandExec := exec.CommandContext(commandCtx, command)
	commandEnv := os.Environ()
	for k, v := range env {
		commandEnv = append(commandEnv, fmt.Sprintf("%s=%s", k, v))
	}
	commandExec.Env = commandEnv

	output, err := commandExec.CombinedOutput()
	if err != nil {
		log.Error(fmt.Sprintf("%s", output))
		return err
	}

	log.Info(fmt.Sprintf("%s", output))
	return nil
}
