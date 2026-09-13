package peer

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"bram/internal/config"
)

type Result struct {
	Output     string
	Stderr     string
	ExitCode   int
	DurationMS int64
}

func Ask(ctx context.Context, spec config.Peer, prompt string) (Result, error) {
	timeout := 2 * time.Minute
	if spec.Timeout != "" {
		parsed, err := time.ParseDuration(spec.Timeout)
		if err != nil {
			return Result{}, err
		}
		timeout = parsed
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, spec.Command, spec.Args...)
	cmd.Stdin = bytes.NewBufferString(prompt)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		Output:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   0,
		DurationMS: time.Since(start).Milliseconds(),
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if ctx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("peer timed out after %s", timeout)
	}
	if err != nil {
		return result, err
	}
	return result, nil
}
