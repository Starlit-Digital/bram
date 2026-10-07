package peer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/cshaiku/bram/internal/config"
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

	stdout := boundedOutput{limit: 2 << 20, cancel: cancel}
	stderr := boundedOutput{limit: 64 << 10, cancel: cancel}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.WaitDelay = time.Second

	err := cmd.Run()
	result := Result{
		Output:     stdout.data.String(),
		Stderr:     stderr.data.String(),
		ExitCode:   0,
		DurationMS: time.Since(start).Milliseconds(),
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if stdout.overflow || stderr.overflow {
		return result, errors.New("peer output exceeded limit (stdout 2 MiB, stderr 64 KiB)")
	}
	if ctx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("peer timed out after %s", timeout)
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

type boundedOutput struct {
	mu       sync.Mutex
	data     bytes.Buffer
	limit    int
	overflow bool
	cancel   context.CancelFunc
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(p) > b.limit-b.data.Len() {
		b.overflow = true
		b.cancel()
		return 0, errors.New("peer output limit")
	}
	return b.data.Write(p)
}
