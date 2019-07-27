package hooks

import (
	"context"
	"fmt"
	"os"
	"time"
	"os/exec"
	"sync"
	"bufio"
	"bytes"

	"github.com/zrepl/zrepl/logger"
)

type contextKey int

const (
	contextKeyLog contextKey = 0
)

type Logger = logger.Logger

func WithLogger(ctx context.Context, log Logger) context.Context {
	return context.WithValue(ctx, contextKeyLog, log)
}

func getLogger(ctx context.Context) Logger {
	if log, ok := ctx.Value(contextKeyLog).(Logger); ok {
		return log
	}
	return logger.NewNullLogger()
}

type logWriter struct {
	mu *sync.Mutex
	buf bytes.Buffer
	scanner *bufio.Scanner
	logFunc func(string)
	level logger.Level
}

func NewLogWriter(mu *sync.Mutex, logFunc func(string)) *logWriter {
	w := new(logWriter)
	w.mu = mu
	w.scanner = bufio.NewScanner(&w.buf)
	w.logFunc = logFunc
	return w
}

func (w *logWriter) Write(in []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	bufCount, bufErr := w.buf.Write(in)
	if bufErr != nil {
		return bufCount, bufErr
	}

	if w.scanner.Scan() {
		w.logFunc(w.scanner.Text())
	}

	return bufCount, nil
}

func (w *logWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.scanner.Scan()
	if w.scanner.Err() == nil && w.scanner.Text() != "" {
		w.logFunc(w.scanner.Text())
	}

	return nil
}


func RunHookCommand(ctx context.Context, command string, env map[string]string, timeout time.Duration) error {
	l := getLogger(ctx)

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmdExec := exec.CommandContext(cmdCtx, command)
	cmdEnv := os.Environ()
	for k, v := range env {
		cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", k, v))
	}
	cmdExec.Env = cmdEnv

	var scanMutex sync.Mutex
	logErrWriter := NewLogWriter(&scanMutex, l.Warn)
	logOutWriter := NewLogWriter(&scanMutex, l.Info)
	defer logErrWriter.Close()
	defer logOutWriter.Close()

	cmdExec.Stderr = logErrWriter
	cmdExec.Stdout = logOutWriter
	err := cmdExec.Start()
	if err != nil {
		l.WithError(err).Error("hook command failed to start")
		return err
	}

	err = cmdExec.Wait()
	if err != nil {
		l.WithError(err).Error("hook command exited with error")
		return err
	}

	return nil
}
