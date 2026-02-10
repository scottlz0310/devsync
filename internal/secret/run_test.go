package secret

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunWithEnv(t *testing.T) {
	t.Run("コマンド未指定はエラー", func(t *testing.T) {
		err := RunWithEnv(nil, map[string]string{"X": "1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "コマンドが指定されていません")
	})

	t.Run("コマンドが見つからない場合はエラー", func(t *testing.T) {
		err := RunWithEnv([]string{"definitely-not-a-real-command-devsync"}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "が見つかりません")
	})

	t.Run("注入した環境変数が子プロセスで参照できる", func(t *testing.T) {
		exe, err := os.Executable()
		require.NoError(t, err)

		// 事前に異なる値を設定しておき、注入値が優先されることも合わせて確認します。
		t.Setenv("DEVSYNC_TEST", "0")

		args := []string{
			exe,
			"-test.run=TestRunWithEnvHelperProcess",
		}

		envVars := map[string]string{
			"DEVSYNC_TEST_HELPER_PROCESS": "1",
			"DEVSYNC_TEST":                "1",
		}

		require.NoError(t, RunWithEnv(args, envVars))
	})

	t.Run("注入値が期待と異なる場合は子プロセスが失敗する", func(t *testing.T) {
		exe, err := os.Executable()
		require.NoError(t, err)

		args := []string{
			exe,
			"-test.run=TestRunWithEnvHelperProcess",
		}

		envVars := map[string]string{
			"DEVSYNC_TEST_HELPER_PROCESS": "1",
			"DEVSYNC_TEST":                "0",
		}

		require.Error(t, RunWithEnv(args, envVars))
	})
}

func TestRunWithEnvHelperProcess(t *testing.T) {
	if os.Getenv("DEVSYNC_TEST_HELPER_PROCESS") != "1" {
		return
	}

	if os.Getenv("DEVSYNC_TEST") != "1" {
		os.Exit(1)
	}

	os.Exit(0)
}
