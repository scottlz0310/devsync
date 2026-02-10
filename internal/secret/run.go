package secret

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunWithEnv は環境変数を注入してコマンドを実行します。
func RunWithEnv(args []string, envVars map[string]string) error {
	if len(args) == 0 {
		return fmt.Errorf("コマンドが指定されていません")
	}

	cmdName := args[0]
	cmdArgs := args[1:]

	// 実行可能ファイルのパスを取得
	cmdPath, err := exec.LookPath(cmdName)
	if err != nil {
		return fmt.Errorf("コマンド '%s' が見つかりません: %w", cmdName, err)
	}

	// 現在の環境変数を取得
	currentEnv := mergeEnv(os.Environ(), envVars)

	// コマンドを準備
	cmd := exec.CommandContext(context.Background(), cmdPath, cmdArgs...)
	cmd.Env = currentEnv
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// コマンドを実行
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func mergeEnv(base []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		return base
	}

	overrideKV := make(map[string]string, len(overrides))
	for key, value := range overrides {
		overrideKV[normalizeEnvKey(key)] = fmt.Sprintf("%s=%s", key, value)
	}

	filtered := make([]string, 0, len(base)+len(overrideKV))
	for _, kv := range base {
		key := kv
		if idx := strings.IndexByte(kv, '='); idx != -1 {
			key = kv[:idx]
		}

		if _, ok := overrideKV[normalizeEnvKey(key)]; ok {
			continue
		}

		filtered = append(filtered, kv)
	}

	for _, kv := range overrideKV {
		filtered = append(filtered, kv)
	}

	return filtered
}
