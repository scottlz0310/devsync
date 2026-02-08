package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/scottlz0310/devsync/internal/secret"
	"github.com/spf13/cobra"
)

func TestRunDaily(t *testing.T) {
	originalUnlock := runUnlockStep
	originalLoadEnv := runLoadEnvStep
	originalSysUpdate := runSysUpdateStep
	originalRepoUpdate := runRepoUpdateStep

	t.Cleanup(func() {
		runUnlockStep = originalUnlock
		runLoadEnvStep = originalLoadEnv
		runSysUpdateStep = originalSysUpdate
		runRepoUpdateStep = originalRepoUpdate
	})

	testCases := []struct {
		name          string
		unlockErr     error
		loadEnvStats  *secret.LoadStats
		loadEnvErr    error
		sysUpdateErr  error
		repoUpdateErr error
		wantErr       bool
		wantErrSubstr string
		wantCalls     []string
	}{
		{
			name:         "全工程成功",
			loadEnvStats: &secret.LoadStats{Loaded: 1},
			wantCalls:    []string{"unlock", "load_env", "sys_update", "repo_update"},
		},
		{
			name:         "環境変数読み込み失敗は継続",
			loadEnvErr:   errors.New("load env failed"),
			loadEnvStats: &secret.LoadStats{},
			wantCalls:    []string{"unlock", "load_env", "sys_update", "repo_update"},
		},
		{
			name:          "アンロック失敗で中断",
			unlockErr:     errors.New("unlock failed"),
			wantErr:       true,
			wantErrSubstr: "unlock failed",
			wantCalls:     []string{"unlock"},
		},
		{
			name:          "sys更新失敗で中断",
			loadEnvStats:  &secret.LoadStats{Loaded: 1},
			sysUpdateErr:  errors.New("sys failed"),
			wantErr:       true,
			wantErrSubstr: "システム更新に失敗しました",
			wantCalls:     []string{"unlock", "load_env", "sys_update"},
		},
		{
			name:          "repo同期失敗で中断",
			loadEnvStats:  &secret.LoadStats{Loaded: 1},
			repoUpdateErr: errors.New("repo failed"),
			wantErr:       true,
			wantErrSubstr: "リポジトリ同期に失敗しました",
			wantCalls:     []string{"unlock", "load_env", "sys_update", "repo_update"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GPAT", "dummy-token")

			calls := make([]string, 0, 4)

			runUnlockStep = func() error {
				calls = append(calls, "unlock")
				return tc.unlockErr
			}

			runLoadEnvStep = func() (*secret.LoadStats, error) {
				calls = append(calls, "load_env")
				return tc.loadEnvStats, tc.loadEnvErr
			}

			runSysUpdateStep = func(*cobra.Command, []string) error {
				calls = append(calls, "sys_update")
				return tc.sysUpdateErr
			}

			runRepoUpdateStep = func(*cobra.Command, []string) error {
				calls = append(calls, "repo_update")
				return tc.repoUpdateErr
			}

			err := runDaily(&cobra.Command{Use: "run"}, nil)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("runDaily() error = nil, want error")
				}

				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("runDaily() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
			} else if err != nil {
				t.Fatalf("runDaily() unexpected error: %v", err)
			}

			if !reflect.DeepEqual(calls, tc.wantCalls) {
				t.Fatalf("runDaily() calls = %#v, want %#v", calls, tc.wantCalls)
			}
		})
	}
}
