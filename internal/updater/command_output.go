package updater

import (
	"fmt"
	"strings"
)

// buildCommandOutputErr はコマンドエラーに出力内容を付加します。
func buildCommandOutputErr(baseErr error, output []byte) error {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return baseErr
	}

	return fmt.Errorf("%w: %s", baseErr, trimmed)
}
