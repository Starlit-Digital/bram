package peer

import (
	"context"
	"strings"
	"testing"

	"github.com/cshaiku/bram/internal/config"
)

func TestAskSendsPromptToCommandStdin(t *testing.T) {
	result, err := Ask(context.Background(), config.Peer{
		Command: "/bin/cat",
		Timeout: "5s",
	}, "hello bram")
	if err != nil {
		t.Fatal(err)
	}
	if result.Output != "hello bram" {
		t.Fatalf("Output = %q, want %q", result.Output, "hello bram")
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestOutputLimitCancelsPeer(t *testing.T) {
	result, err := Ask(context.Background(), config.Peer{Command: "/bin/sh", Args: []string{"-c", "while :; do printf '" + strings.Repeat("x", 4096) + "'; done"}, Timeout: "5s"}, "synthetic")
	if err == nil || !strings.Contains(err.Error(), "exceeded limit") || len(result.Output) > 2<<20 {
		t.Fatalf("bytes=%d err=%v", len(result.Output), err)
	}
}

func TestAskReportsMissingCommand(t *testing.T) {
	_, err := Ask(context.Background(), config.Peer{
		Command: "bram-definitely-missing-command",
		Timeout: "5s",
	}, "hello")
	if err == nil {
		t.Fatal("expected missing command error")
	}
}
