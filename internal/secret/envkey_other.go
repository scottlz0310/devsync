//go:build !windows

package secret

func normalizeEnvKey(key string) string {
	return key
}
