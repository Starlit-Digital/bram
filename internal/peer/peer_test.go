package peer

import (
	"context"
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

func TestAskReportsMissingCommand(t *testing.T) {
	_, err := Ask(context.Background(), config.Peer{
		Command: "bram-definitely-missing-command",
		Timeout: "5s",
	}, "hello")
	if err == nil {
		t.Fatal("expected missing command error")
	}
}
