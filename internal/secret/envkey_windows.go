//go:build windows

package secret

import "strings"

func normalizeEnvKey(key string) string {
	// Windows の環境変数名は基本的に大文字小文字を区別しないため、正規化して比較します。
	return strings.ToUpper(key)
}
