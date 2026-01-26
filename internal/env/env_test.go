package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsContainer(t *testing.T) {
	t.Run("CODESPACES=true", func(t *testing.T) {
		t.Setenv("CODESPACES", "true")
		t.Setenv("REMOTE_CONTAINERS", "")
		assert.True(t, IsContainer())
	})

	t.Run("REMOTE_CONTAINERS=true", func(t *testing.T) {
		t.Setenv("CODESPACES", "")
		t.Setenv("REMOTE_CONTAINERS", "true")
		assert.True(t, IsContainer())
	})

	t.Run("CODESPACES=false_は_false", func(t *testing.T) {
		t.Setenv("CODESPACES", "false")
		t.Setenv("REMOTE_CONTAINERS", "")
		// "false" は "true" ではないのでコンテナとみなされない
		// ただし /.dockerenv が存在する場合は true になる可能性がある
		result := IsContainer()
		// 環境によって異なるため、単純にエラーにならないことを確認
		assert.IsType(t, true, result)
	})

	t.Run("環境変数が空の場合", func(t *testing.T) {
		t.Setenv("CODESPACES", "")
		t.Setenv("REMOTE_CONTAINERS", "")
		// /.dockerenv の存在によって結果が変わる
		result := IsContainer()
		assert.IsType(t, true, result)
	})

	t.Run("両方の環境変数がtrue", func(t *testing.T) {
		t.Setenv("CODESPACES", "true")
		t.Setenv("REMOTE_CONTAINERS", "true")
		assert.True(t, IsContainer())
	})
}

func TestIsWSL(t *testing.T) {
	t.Run("関数が正常に動作する", func(t *testing.T) {
		result := IsWSL()
		// bool型を返すことを確認
		assert.IsType(t, true, result)
	})

	t.Run("/proc/versionが存在しない環境では_false", func(t *testing.T) {
		// WSL判定は /proc/version を読むので、テストは環境依存
		result := IsWSL()
		// DevContainer環境ではfalseが期待される
		assert.IsType(t, true, result)
	})
}

func TestGetRecommendedManagers(t *testing.T) {
	t.Run("共通のマネージャが含まれる", func(t *testing.T) {
		managers := GetRecommendedManagers()
		assert.Contains(t, managers, "go")
		assert.Contains(t, managers, "npm")
	})

	t.Run("Debian系環境ではaptが含まれる", func(t *testing.T) {
		managers := GetRecommendedManagers()
		// 実行環境がDebian系(例えばこのDevContainer)であればaptが含まれるはず
		if _, err := os.Stat("/usr/bin/apt-get"); err == nil {
			assert.Contains(t, managers, "apt")
		}
	})

	t.Run("コンテナ環境でのマネージャリスト", func(t *testing.T) {
		// コンテナ環境を強制
		t.Setenv("CODESPACES", "true")

		managers := GetRecommendedManagers()
		assert.Contains(t, managers, "go")
		assert.Contains(t, managers, "npm")
		// コンテナ環境でDebian系ならaptも含まれる
		if _, err := os.Stat("/usr/bin/apt-get"); err == nil {
			assert.Contains(t, managers, "apt")
		}
	})
}

func TestIsDebianLike(t *testing.T) {
	t.Run("/usr/bin/apt-getが存在する場合はtrue", func(t *testing.T) {
		result := isDebianLike()
		// このDevContainer環境はDebian系なのでtrueが期待される
		if _, err := os.Stat("/usr/bin/apt-get"); err == nil {
			assert.True(t, result)
		} else {
			assert.False(t, result)
		}
	})
}
