package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"golang.org/x/term"
)

// MisuseError represents incorrect command usage (invalid flag values, wrong args, etc.).
// Commands that detect misuse should return this type so main can exit with code 2.
type MisuseError struct{ msg string }

func (e *MisuseError) Error() string { return e.msg }

func newMisuseError(format string, args ...any) error {
	return &MisuseError{msg: fmt.Sprintf(format, args...)}
}

// enteredPreRun is set true once PersistentPreRun fires, meaning cobra accepted the
// command (flags parsed, arg count valid). Errors before this point are misuse (exit 2).
var enteredPreRun bool

// isStdinTTY reports whether os.Stdin is an interactive terminal.
func isStdinTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// Global logger
var logger *slog.Logger

func initLogger() {
	logLevel := slog.LevelInfo
	switch {
	case quiet:
		logLevel = slog.LevelError
	case verbose:
		logLevel = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})

	logger = slog.New(handler)
	slog.SetDefault(logger)
}

// exitCodeForError returns the appropriate exit code for err.
//
// Exit 2 is used when the error is a MisuseError or when cobra never entered
// PersistentPreRun (meaning it rejected the command before execution — wrong
// arg count, unknown flag, etc.). Exit 1 is used for all other errors.
func exitCodeForError(err error, preRunEntered bool) int {
	if err == nil {
		return 0
	}
	if _, ok := errors.AsType[*MisuseError](err); !preRunEntered || ok {
		return 2
	}
	return 1
}

func main() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCodeForError(err, enteredPreRun))
	}
}
