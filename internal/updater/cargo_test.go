package updater

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCargoUpdater_parseInstallList(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []PackageInfo
	}{
		{
			name:     "空の出力",
			output:   "",
			expected: nil,
		},
		{
			name: "単一パッケージ",
			output: `ripgrep v13.0.0:
    rg
`,
			expected: []PackageInfo{
				{Name: "ripgrep", CurrentVersion: "13.0.0"},
			},
		},
		{
			name: "複数パッケージ",
			output: `ripgrep v13.0.0:
    rg

cargo-update v16.0.0:
    cargo-install-update
    cargo-install-update-config
`,
			expected: []PackageInfo{
				{Name: "ripgrep", CurrentVersion: "13.0.0"},
				{Name: "cargo-update", CurrentVersion: "16.0.0"},
			},
		},
		{
			name: "不正な行は無視される",
			output: `no-colon-line
pkg-only:
pkg 1.2.3
pkg 1.2.3:
    bin
`,
			expected: []PackageInfo{
				{Name: "pkg", CurrentVersion: "1.2.3"},
			},
		},
		{
			name: "versionにvが無い場合も許容",
			output: `pkg 1.2.3:
    bin
`,
			expected: []PackageInfo{
				{Name: "pkg", CurrentVersion: "1.2.3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CargoUpdater{}
			got := c.parseInstallList(tt.output)

			assert.Len(t, got, len(tt.expected))

			for i := range tt.expected {
				assert.Equal(t, tt.expected[i].Name, got[i].Name)
				assert.Equal(t, tt.expected[i].CurrentVersion, got[i].CurrentVersion)
				assert.Equal(t, "", got[i].NewVersion)
			}
		})
	}
}
