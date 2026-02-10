package main

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestRunGhOutputWithRetry_RateLimitThenSuccess(t *testing.T) {
	originalCommandStep := repoExecCommandStep
	originalSleepStep := ghSleepStep
	t.Cleanup(func() {
		repoExecCommandStep = originalCommandStep
		ghSleepStep = originalSleepStep
	})

	var calls int
	repoExecCommandStep = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		calls++

		if calls == 1 {
			return helperProcessCommand(ctx, "", "exceeded retry limit, last status: 429 Too Many Requests, request id: 50e58657-3180-4fd7-99f4-e0d005d07a9d\n", 1)
		}

		return helperProcessCommand(ctx, "[]\n", "", 0)
	}

	var slept []time.Duration
	ghSleepStep = func(ctx context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}

	got, stderr, err := runGhOutputWithRetry(context.Background(), "", "repo", "list", "owner", "--limit", "1", "--json", "name")
	if err != nil {
		t.Fatalf("runGhOutputWithRetry() error = %v", err)
	}

	if string(got) != "[]\n" {
		t.Fatalf("stdout = %q, want %q", string(got), "[]\n")
	}

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}

	if len(slept) != 1 {
		t.Fatalf("slept len = %d, want 1. slept=%v", len(slept), slept)
	}
}

func TestRunGhOutputWithRetry_NonRetryableErrorDoesNotRetry(t *testing.T) {
	originalCommandStep := repoExecCommandStep
	originalSleepStep := ghSleepStep
	t.Cleanup(func() {
		repoExecCommandStep = originalCommandStep
		ghSleepStep = originalSleepStep
	})

	var calls int
	repoExecCommandStep = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		calls++
		return helperProcessCommand(ctx, "", "auth failed\n", 1)
	}

	var slept []time.Duration
	ghSleepStep = func(ctx context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}

	_, _, err := runGhOutputWithRetry(context.Background(), "", "repo", "list", "owner", "--limit", "1", "--json", "name")
	if err == nil {
		t.Fatalf("runGhOutputWithRetry() error = nil, want error")
	}

	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}

	if len(slept) != 0 {
		t.Fatalf("sleep should not be called. slept=%v", slept)
	}
}

func TestCalcGhRetryDelay_ParsesRetryAfter(t *testing.T) {
	t.Parallel()

	got := calcGhRetryDelay(1, "Retry-After: 10")
	// parseRetryAfter() で 10s を見つけた場合は +1s して返す
	if got != 11*time.Second {
		t.Fatalf("calcGhRetryDelay() = %v, want %v", got, 11*time.Second)
	}
}
