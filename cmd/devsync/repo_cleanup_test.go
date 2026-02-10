package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestWantsCleanupTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		targets  []string
		want     string
		expected bool
	}{
		{
			name:     "空のtargets",
			targets:  nil,
			want:     "merged",
			expected: false,
		},
		{
			name:     "一致",
			targets:  []string{"merged"},
			want:     "merged",
			expected: true,
		},
		{
			name:     "大文字小文字と空白を無視",
			targets:  []string{"  SQUASHED  "},
			want:     "squashed",
			expected: true,
		},
		{
			name:     "含まれない",
			targets:  []string{"merged"},
			want:     "squashed",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := wantsCleanupTarget(tt.targets, tt.want)
			if got != tt.expected {
				t.Fatalf("wantsCleanupTarget(%#v, %q) = %v, want %v", tt.targets, tt.want, got, tt.expected)
			}
		})
	}
}

func TestListMergedPRHeads(t *testing.T) {
	originalLookPathStep := repoLookPathStep
	originalCommandStep := repoExecCommandStep
	t.Cleanup(func() {
		repoLookPathStep = originalLookPathStep
		repoExecCommandStep = originalCommandStep
	})

	t.Run("ghコマンドがない場合は文脈付きエラー", func(t *testing.T) {
		repoPath := t.TempDir()

		repoLookPathStep = func(string) (string, error) {
			return "", errors.New("not found")
		}

		repoExecCommandStep = func(context.Context, string, ...string) *exec.Cmd {
			t.Fatalf("repoExecCommandStep should not be called when gh is missing")

			return nil
		}

		_, err := listMergedPRHeads(context.Background(), repoPath, "main")
		if err == nil {
			t.Fatalf("listMergedPRHeads() error = nil, want error")
		}

		if !strings.Contains(err.Error(), "gh コマンドが見つかりません") {
			t.Fatalf("error should contain missing gh message: %v", err)
		}
	})

	t.Run("gh実行失敗時はstderrを含む", func(t *testing.T) {
		repoPath := t.TempDir()

		repoLookPathStep = func(file string) (string, error) {
			if file != "gh" {
				t.Fatalf("repoLookPathStep file = %q, want gh", file)
			}

			return "/usr/bin/gh", nil
		}

		repoExecCommandStep = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			assertGHPullRequestListArgs(t, name, arg, "main")

			return helperProcessCommand(ctx, "", "auth failed\n", 1)
		}

		_, err := listMergedPRHeads(context.Background(), repoPath, "main")
		if err == nil {
			t.Fatalf("listMergedPRHeads() error = nil, want error")
		}

		if !strings.Contains(err.Error(), "gh pr list の実行に失敗しました") {
			t.Fatalf("error should contain command failure message: %v", err)
		}

		if !strings.Contains(err.Error(), "auth failed") {
			t.Fatalf("error should contain stderr details: %v", err)
		}
	})

	t.Run("JSON解析失敗時はエラー", func(t *testing.T) {
		repoPath := t.TempDir()

		repoLookPathStep = func(string) (string, error) {
			return "/usr/bin/gh", nil
		}

		repoExecCommandStep = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			assertGHPullRequestListArgs(t, name, arg, "main")

			return helperProcessCommand(ctx, "not json", "", 0)
		}

		_, err := listMergedPRHeads(context.Background(), repoPath, "main")
		if err == nil {
			t.Fatalf("listMergedPRHeads() error = nil, want error")
		}

		if !strings.Contains(err.Error(), "PR 一覧の解析に失敗") {
			t.Fatalf("error should contain json unmarshal message: %v", err)
		}
	})

	t.Run("正常時は最新のheadRefOidを返す", func(t *testing.T) {
		repoPath := t.TempDir()

		repoLookPathStep = func(string) (string, error) {
			return "/usr/bin/gh", nil
		}

		repoExecCommandStep = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			assertGHPullRequestListArgs(t, name, arg, "main")

			stdout := `[{"headRefName":" feature/a ","headRefOid":"111","mergedAt":"2026-02-09T00:00:00Z"},` +
				`{"headRefName":"feature/a","headRefOid":" 222 ","mergedAt":"2026-02-10T00:00:00Z"},` +
				`{"headRefName":"feature/b","headRefOid":"333","mergedAt":"2026-02-08T00:00:00Z"},` +
				`{"headRefName":"","headRefOid":"444","mergedAt":"2026-02-10T00:00:00Z"},` +
				`{"headRefName":"feature/c","headRefOid":"","mergedAt":"2026-02-10T00:00:00Z"}]`

			return helperProcessCommand(ctx, stdout+"\n", "", 0)
		}

		got, err := listMergedPRHeads(context.Background(), repoPath, "main")
		if err != nil {
			t.Fatalf("listMergedPRHeads() unexpected error: %v", err)
		}

		want := mergedPRHeadsResult{
			Heads: map[string]string{
				"feature/a": "222",
				"feature/b": "333",
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("listMergedPRHeads() = %#v, want %#v", got, want)
		}
	})
}

func helperProcessCommand(ctx context.Context, stdout, stderr string, exitCode int) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")

	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"DEVSYNC_HELPER_STDOUT="+stdout,
		"DEVSYNC_HELPER_STDERR="+stderr,
		"DEVSYNC_HELPER_EXIT_CODE="+strconv.Itoa(exitCode),
	)

	return cmd
}

func assertGHPullRequestListArgs(t *testing.T, name string, gotArgs []string, baseBranch string) {
	t.Helper()

	if name != "gh" {
		t.Fatalf("repoExecCommandStep name = %q, want gh", name)
	}

	wantArgs := []string{
		"pr",
		"list",
		"--state",
		"merged",
		"--base",
		baseBranch,
		"--limit",
		strconv.Itoa(githubRepoListLimit),
		"--json",
		"headRefName,headRefOid,mergedAt",
	}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("repoExecCommandStep args = %#v, want %#v", gotArgs, wantArgs)
	}
}
