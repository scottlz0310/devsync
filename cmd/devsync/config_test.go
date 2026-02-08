package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/scottlz0310/devsync/internal/config"
)

func TestNormalizeRepoRoot(t *testing.T) {
	testCases := []struct {
		name      string
		setup     func(t *testing.T)
		input     string
		want      string
		expectErr bool
	}{
		{
			name:      "空文字はエラー",
			input:     "   ",
			expectErr: true,
		},
		{
			name:  "通常パスはクリーン化される",
			input: "/tmp/work/../work/src",
			want:  "/tmp/work/src",
		},
		{
			name: "チルダ展開",
			setup: func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
			},
			input: "~/src",
			want:  "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}

			got, err := normalizeRepoRoot(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("normalizeRepoRoot(%q) error = nil, want error", tc.input)
				}

				return
			}

			if err != nil {
				t.Fatalf("normalizeRepoRoot(%q) unexpected error: %v", tc.input, err)
			}

			want := tc.want
			if tc.input == "~/src" {
				home, homeErr := os.UserHomeDir()
				if homeErr != nil {
					t.Fatalf("os.UserHomeDir() unexpected error: %v", homeErr)
				}

				want = filepath.Join(home, "src")
			}

			if got != filepath.Clean(want) {
				t.Fatalf("normalizeRepoRoot(%q) = %q, want %q", tc.input, got, filepath.Clean(want))
			}
		})
	}
}

func TestEnsureRepoRoot(t *testing.T) {
	t.Parallel()

	t.Run("既存ディレクトリはそのまま成功", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		called := false

		err := ensureRepoRoot(dir, func(path string) (bool, error) {
			called = true
			return false, nil
		})
		if err != nil {
			t.Fatalf("ensureRepoRoot() unexpected error: %v", err)
		}

		if called {
			t.Fatalf("confirmCreate should not be called for existing directory")
		}
	})

	t.Run("既存ファイルはエラー", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()

		filePath := filepath.Join(root, "repo-root-file")
		if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		err := ensureRepoRoot(filePath, func(path string) (bool, error) {
			return true, nil
		})
		if err == nil {
			t.Fatalf("ensureRepoRoot() error = nil, want error")
		}
	})

	t.Run("未存在ディレクトリで作成承認時は作成して成功", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		target := filepath.Join(root, "new-src")

		err := ensureRepoRoot(target, func(path string) (bool, error) {
			if path != target {
				t.Fatalf("confirm path = %q, want %q", path, target)
			}

			return true, nil
		})
		if err != nil {
			t.Fatalf("ensureRepoRoot() unexpected error: %v", err)
		}

		info, statErr := os.Stat(target)
		if statErr != nil {
			t.Fatalf("created directory stat error: %v", statErr)
		}

		if !info.IsDir() {
			t.Fatalf("created path is not directory: %s", target)
		}
	})

	t.Run("未存在ディレクトリで拒否時はキャンセル終了", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		target := filepath.Join(root, "new-src")

		err := ensureRepoRoot(target, func(path string) (bool, error) {
			return false, nil
		})
		if !errors.Is(err, errConfigInitCanceled) {
			t.Fatalf("ensureRepoRoot() error = %v, want errConfigInitCanceled", err)
		}

		if _, statErr := os.Stat(target); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("target should not exist after cancel, statErr=%v", statErr)
		}
	})
}

func TestResolveGitHubOwnerDefault(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		lookup func(context.Context) (string, error)
		want   string
	}{
		{
			name: "取得成功時はtrimした値を返す",
			lookup: func(context.Context) (string, error) {
				return "scottlz0310\n", nil
			},
			want: "scottlz0310",
		},
		{
			name: "空白のみは空文字を返す",
			lookup: func(context.Context) (string, error) {
				return "   \n", nil
			},
			want: "",
		},
		{
			name: "取得失敗時は空文字を返す",
			lookup: func(context.Context) (string, error) {
				return "", errors.New("lookup failed")
			},
			want: "",
		},
		{
			name:   "lookup未指定時は空文字を返す",
			lookup: nil,
			want:   "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolveGitHubOwnerDefault(context.Background(), tc.lookup)
			if got != tc.want {
				t.Fatalf("resolveGitHubOwnerDefault() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildConfigInitDefaults(t *testing.T) {
	t.Parallel()

	promptOptions := []string{"apt", "brew", "go", "npm", "snap", "pipx", "cargo"}

	testCases := []struct {
		name                string
		home                string
		recommendedManagers []string
		existingCfg         *config.Config
		autoOwner           string
		want                configInitDefaults
	}{
		{
			name:                "既存設定なしは推奨値と自動オーナーを使用",
			home:                "/home/dev",
			recommendedManagers: []string{"apt", "snap"},
			existingCfg:         nil,
			autoOwner:           "auto-user",
			want: configInitDefaults{
				RepoRoot:        "/home/dev/src",
				GitHubOwner:     "auto-user",
				Concurrency:     8,
				EnabledManagers: []string{"apt", "snap"},
			},
		},
		{
			name:                "既存設定がある場合は既存値を優先",
			home:                "/home/dev",
			recommendedManagers: []string{"brew", "go"},
			existingCfg: &config.Config{
				Control: config.ControlConfig{
					Concurrency: 16,
				},
				Repo: config.RepoConfig{
					Root: "/work/repos",
					GitHub: config.GitHubConfig{
						Owner: "my-org",
					},
				},
				Sys: config.SysConfig{
					Enable: []string{"apt", "unknown", "apt", "npm"},
				},
			},
			autoOwner: "auto-user",
			want: configInitDefaults{
				RepoRoot:        "/work/repos",
				GitHubOwner:     "my-org",
				Concurrency:     16,
				EnabledManagers: []string{"apt", "npm"},
			},
		},
		{
			name:                "既存設定のenableが空なら推奨値へフォールバック",
			home:                "/home/dev",
			recommendedManagers: []string{"brew", "not-supported"},
			existingCfg: &config.Config{
				Control: config.ControlConfig{
					Concurrency: 0,
				},
				Repo: config.RepoConfig{
					Root: "/repos",
				},
				Sys: config.SysConfig{
					Enable: []string{},
				},
			},
			autoOwner: "auto-user",
			want: configInitDefaults{
				RepoRoot:        "/repos",
				GitHubOwner:     "auto-user",
				Concurrency:     8,
				EnabledManagers: []string{"brew"},
			},
		},
		{
			name:                "候補が空の場合はprompt options全体を使用",
			home:                "/home/dev",
			recommendedManagers: nil,
			existingCfg: &config.Config{
				Sys: config.SysConfig{
					Enable: []string{"unknown"},
				},
			},
			autoOwner: "",
			want: configInitDefaults{
				RepoRoot:        "/home/dev/src",
				GitHubOwner:     "",
				Concurrency:     8,
				EnabledManagers: promptOptions,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildConfigInitDefaults(tc.home, tc.recommendedManagers, promptOptions, tc.existingCfg, tc.autoOwner)

			if got.RepoRoot != tc.want.RepoRoot {
				t.Fatalf("RepoRoot = %q, want %q", got.RepoRoot, tc.want.RepoRoot)
			}

			if got.GitHubOwner != tc.want.GitHubOwner {
				t.Fatalf("GitHubOwner = %q, want %q", got.GitHubOwner, tc.want.GitHubOwner)
			}

			if got.Concurrency != tc.want.Concurrency {
				t.Fatalf("Concurrency = %d, want %d", got.Concurrency, tc.want.Concurrency)
			}

			if !reflect.DeepEqual(got.EnabledManagers, tc.want.EnabledManagers) {
				t.Fatalf("EnabledManagers = %#v, want %#v", got.EnabledManagers, tc.want.EnabledManagers)
			}
		})
	}
}

func TestGeneratedShellScripts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		buildScript     func(exePath string) string
		requiredPhrases []string
	}{
		{
			name:        "bashスクリプトはアンロック・env読込・run呼び出しを含む",
			buildScript: getBashScript,
			requiredPhrases: []string{
				`command -v devsync`,
				`token="$(bw unlock --raw)"`,
				`env_output="$("$DEVSYNC_PATH" env export)"`,
				`if [ $status -ne 0 ]; then`,
				`devsync-unlock || return 1`,
				`devsync-load-env || return 1`,
				`"$DEVSYNC_PATH" run "$@"`,
			},
		},
		{
			name:        "zshスクリプトはアンロック・env読込・run呼び出しを含む",
			buildScript: getZshScript,
			requiredPhrases: []string{
				`command -v devsync`,
				`token="$(bw unlock --raw)"`,
				`env_output="$("$DEVSYNC_PATH" env export)"`,
				`if [[ $status -ne 0 ]]; then`,
				`devsync-unlock || return 1`,
				`devsync-load-env || return 1`,
				`"$DEVSYNC_PATH" run "$@"`,
			},
		},
		{
			name:        "PowerShellスクリプトはアンロック・env読込・run呼び出しを含む",
			buildScript: getPowerShellScript,
			requiredPhrases: []string{
				`Get-Command devsync`,
				`$token = & bw unlock --raw`,
				`$envExports = & $DEVSYNC_PATH env export`,
				`if ($LASTEXITCODE -ne 0) { return $LASTEXITCODE }`,
				`Invoke-Expression -Command $envExports -ErrorAction Stop`,
				`Write-Error "環境変数の読み込み中にエラーが発生しました: $_"`,
				`devsync-unlock`,
				`devsync-load-env`,
				`& $DEVSYNC_PATH run @args`,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			script := tc.buildScript("/tmp/devsync")
			for _, phrase := range tc.requiredPhrases {
				if !strings.Contains(script, phrase) {
					t.Fatalf("generated script does not contain required phrase %q", phrase)
				}
			}
		})
	}
}
