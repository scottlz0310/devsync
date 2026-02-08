package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/scottlz0310/devsync/internal/config"
	repomgr "github.com/scottlz0310/devsync/internal/repo"
)

func TestResolveRepoJobs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		configJobs int
		flagJobs   int
		want       int
	}{
		{
			name:       "フラグ優先",
			configJobs: 8,
			flagJobs:   3,
			want:       3,
		},
		{
			name:       "フラグ未指定なら設定値",
			configJobs: 6,
			flagJobs:   0,
			want:       6,
		},
		{
			name:       "設定が不正なら1",
			configJobs: 0,
			flagJobs:   0,
			want:       1,
		},
		{
			name:       "負数フラグは設定値にフォールバック",
			configJobs: 5,
			flagJobs:   -1,
			want:       5,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolveRepoJobs(tc.configJobs, tc.flagJobs)
			if got != tc.want {
				t.Fatalf("resolveRepoJobs(%d, %d) = %d, want %d", tc.configJobs, tc.flagJobs, got, tc.want)
			}
		})
	}
}

func TestResolveRepoSubmoduleUpdate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		configValue     bool
		enableOverride  bool
		disableOverride bool
		want            bool
		expectErr       bool
	}{
		{
			name:            "上書きなしは設定値を採用",
			configValue:     true,
			enableOverride:  false,
			disableOverride: false,
			want:            true,
			expectErr:       false,
		},
		{
			name:            "有効化上書き",
			configValue:     false,
			enableOverride:  true,
			disableOverride: false,
			want:            true,
			expectErr:       false,
		},
		{
			name:            "無効化上書き",
			configValue:     true,
			enableOverride:  false,
			disableOverride: true,
			want:            false,
			expectErr:       false,
		},
		{
			name:            "矛盾指定はエラー",
			configValue:     true,
			enableOverride:  true,
			disableOverride: true,
			want:            false,
			expectErr:       true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveRepoSubmoduleUpdate(tc.configValue, tc.enableOverride, tc.disableOverride)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("resolveRepoSubmoduleUpdate() error = nil, want error")
				}

				return
			}

			if err != nil {
				t.Fatalf("resolveRepoSubmoduleUpdate() unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("resolveRepoSubmoduleUpdate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBuildRepoJobDisplayName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		root     string
		repoPath string
		want     string
	}{
		{
			name:     "root直下は相対パス",
			root:     "/work/src",
			repoPath: "/work/src/devsync",
			want:     "devsync",
		},
		{
			name:     "ネストしたパスは相対表示",
			root:     "/work/src",
			repoPath: "/work/src/team-a/api",
			want:     "team-a/api",
		},
		{
			name:     "root自身はドット表示",
			root:     "/work/src",
			repoPath: "/work/src",
			want:     ".",
		},
		{
			name:     "root外はベース名表示",
			root:     "/work/src",
			repoPath: "/opt/repos/sample",
			want:     "sample",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildRepoJobDisplayName(tc.root, tc.repoPath)
			if got != tc.want {
				t.Fatalf("buildRepoJobDisplayName(%q, %q) = %q, want %q", tc.root, tc.repoPath, got, tc.want)
			}
		})
	}
}

func TestWrapRepoRootError(t *testing.T) {
	t.Parallel()

	notFoundErr := fmt.Errorf("ルートディレクトリにアクセスできません: %w", os.ErrNotExist)

	testCases := []struct {
		name           string
		err            error
		root           string
		rootOverridden bool
		configExists   bool
		configPath     string
		wantHint       bool
	}{
		{
			name:           "設定未初期化なら config init を案内",
			err:            notFoundErr,
			root:           "/tmp/src",
			rootOverridden: false,
			configExists:   false,
			configPath:     "/tmp/.config/devsync/config.yaml",
			wantHint:       true,
		},
		{
			name:           "設定ファイルがあれば案内しない",
			err:            notFoundErr,
			root:           "/tmp/src",
			rootOverridden: false,
			configExists:   true,
			configPath:     "/tmp/.config/devsync/config.yaml",
			wantHint:       false,
		},
		{
			name:           "root上書き時は案内しない",
			err:            notFoundErr,
			root:           "/tmp/src",
			rootOverridden: true,
			configExists:   false,
			configPath:     "/tmp/.config/devsync/config.yaml",
			wantHint:       false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			wrapped := wrapRepoRootError(tc.err, tc.root, tc.rootOverridden, tc.configExists, tc.configPath)

			if hasHint := strings.Contains(wrapped.Error(), "devsync config init"); hasHint != tc.wantHint {
				t.Fatalf("wrapRepoRootError() hint = %v, want %v. got=%q", hasHint, tc.wantHint, wrapped.Error())
			}
		})
	}
}

func TestWriteRepoTable(t *testing.T) {
	t.Parallel()

	repos := []repomgr.Info{
		{
			Name:        "devsync-manual",
			Status:      repomgr.StatusDirty,
			Ahead:       1,
			HasUpstream: true,
			Path:        "/home/dev/src/devsync-manual",
		},
		{
			Name:        "devsync-no-upstream",
			Status:      repomgr.StatusNoUpstream,
			Ahead:       0,
			HasUpstream: false,
			Path:        "/home/dev/src/devsync-no-upstream",
		},
	}

	var output bytes.Buffer
	if err := writeRepoTable(&output, repos); err != nil {
		t.Fatalf("writeRepoTable() unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) < 4 {
		t.Fatalf("unexpected table output lines: %q", output.String())
	}

	dataLines := lines[2:]
	for _, line := range dataLines {
		if strings.Contains(line, "1/home/") || strings.Contains(line, "-/home/") {
			t.Fatalf("Ahead列とパス列が結合されています: %q", line)
		}

		fields := strings.Fields(line)
		if len(fields) != 4 {
			t.Fatalf("table row fields = %d, want 4. line=%q", len(fields), line)
		}
	}
}

func TestSelectRepoCloneURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		protocol string
		repo     githubRepo
		want     string
	}{
		{
			name:     "https優先",
			protocol: "https",
			repo: githubRepo{
				URL:    "https://github.com/a/b.git",
				SSHURL: "git@github.com:a/b.git",
			},
			want: "https://github.com/a/b.git",
		},
		{
			name:     "ssh優先",
			protocol: "ssh",
			repo: githubRepo{
				URL:    "https://github.com/a/b.git",
				SSHURL: "git@github.com:a/b.git",
			},
			want: "git@github.com:a/b.git",
		},
		{
			name:     "ssh指定でもsshURLがなければhttpsへフォールバック",
			protocol: "ssh",
			repo: githubRepo{
				URL: "https://github.com/a/b.git",
			},
			want: "https://github.com/a/b.git",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := selectRepoCloneURL(tc.protocol, tc.repo)
			if got != tc.want {
				t.Fatalf("selectRepoCloneURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBootstrapReposFromGitHub(t *testing.T) {
	originalListStep := repoListGitHubReposStep
	originalCloneStep := repoCloneRepoStep

	t.Cleanup(func() {
		repoListGitHubReposStep = originalListStep
		repoCloneRepoStep = originalCloneStep
	})

	t.Run("owner未設定時は処理しない", func(t *testing.T) {
		listCalled := false
		repoListGitHubReposStep = func(ctx context.Context, owner string) ([]githubRepo, error) {
			listCalled = true
			return nil, nil
		}

		got, err := bootstrapReposFromGitHub(context.Background(), t.TempDir(), &config.Config{
			Repo: config.RepoConfig{
				GitHub: config.GitHubConfig{Owner: ""},
			},
		}, false)
		if err != nil {
			t.Fatalf("bootstrapReposFromGitHub() unexpected error: %v", err)
		}

		if listCalled {
			t.Fatalf("repo list step should not be called when owner is empty")
		}

		if len(got.ReadyPaths) != 0 || got.PlannedOnly != 0 {
			t.Fatalf("unexpected bootstrap result: %#v", got)
		}
	})

	t.Run("dry-runでclone計画のみ作成", func(t *testing.T) {
		root := t.TempDir()

		existingRepo := filepath.Join(root, "exists")
		if err := os.MkdirAll(filepath.Join(existingRepo, ".git"), 0o755); err != nil {
			t.Fatalf("failed to setup existing repo: %v", err)
		}

		repoListGitHubReposStep = func(ctx context.Context, owner string) ([]githubRepo, error) {
			return []githubRepo{
				{Name: "exists", URL: "https://github.com/a/exists.git"},
				{Name: "new-repo", URL: "https://github.com/a/new-repo.git"},
				{Name: "archived", URL: "https://github.com/a/archived.git", IsArchived: true},
			}, nil
		}

		cloneCalled := false
		repoCloneRepoStep = func(ctx context.Context, cloneURL, targetPath string) error {
			cloneCalled = true
			return nil
		}

		got, err := bootstrapReposFromGitHub(context.Background(), root, &config.Config{
			Repo: config.RepoConfig{
				GitHub: config.GitHubConfig{
					Owner:    "owner",
					Protocol: "https",
				},
			},
		}, true)
		if err != nil {
			t.Fatalf("bootstrapReposFromGitHub() unexpected error: %v", err)
		}

		wantReady := []string{filepath.Join(root, "exists")}
		if !reflect.DeepEqual(got.ReadyPaths, wantReady) {
			t.Fatalf("ReadyPaths = %#v, want %#v", got.ReadyPaths, wantReady)
		}

		if got.PlannedOnly != 1 {
			t.Fatalf("PlannedOnly = %d, want 1", got.PlannedOnly)
		}

		if cloneCalled {
			t.Fatalf("clone step should not be called in dry-run mode")
		}
	})
}
