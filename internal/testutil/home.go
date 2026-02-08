package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// SetTestHome はテスト内で os.UserHomeDir() の参照先を確実に切り替えるためのヘルパーです。
// Windows では HOME ではなく USERPROFILE が参照されるため、両方を設定します。
func SetTestHome(t *testing.T, home string) {
	t.Helper()

	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	// Go の実装は USERPROFILE を優先しますが、念のため従来の変数も合わせて揃えます。
	if runtime.GOOS != "windows" {
		return
	}

	vol := filepath.VolumeName(home)
	if vol == "" {
		return
	}

	rest := home[len(vol):]
	if rest == "" {
		rest = string(os.PathSeparator)
	}

	t.Setenv("HOMEDRIVE", vol)
	t.Setenv("HOMEPATH", rest)
}
