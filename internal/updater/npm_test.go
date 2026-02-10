package updater

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNpmUpdater_parseOutdatedJSON(t *testing.T) {
	tests := []struct {
		name     string
		output   []byte
		expected map[string]PackageInfo
	}{
		{
			name:     "空の出力",
			output:   nil,
			expected: nil,
		},
		{
			name:     "不正なJSON",
			output:   []byte("{not-json"),
			expected: nil,
		},
		{
			name: "更新可能パッケージあり",
			output: []byte(`{
  "typescript": { "current": "5.1.0", "wanted": "5.1.0", "latest": "5.2.0", "location": "/usr/local/lib" },
  "@scope/pkg": { "current": "1.0.0", "wanted": "1.0.1", "latest": "1.1.0", "location": "/usr/local/lib" }
}`),
			expected: map[string]PackageInfo{
				"typescript": {
					Name:           "typescript",
					CurrentVersion: "5.1.0",
					NewVersion:     "5.2.0",
				},
				"@scope/pkg": {
					Name:           "@scope/pkg",
					CurrentVersion: "1.0.0",
					NewVersion:     "1.1.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NpmUpdater{}
			got := n.parseOutdatedJSON(tt.output)

			if tt.expected == nil {
				assert.Empty(t, got)
				return
			}

			assert.Len(t, got, len(tt.expected))

			gotMap := make(map[string]PackageInfo, len(got))
			for _, pkg := range got {
				gotMap[pkg.Name] = pkg
			}

			for name, expectedPkg := range tt.expected {
				pkg, ok := gotMap[name]
				assert.True(t, ok, "package %q が見つかりません", name)
				assert.Equal(t, expectedPkg.CurrentVersion, pkg.CurrentVersion)
				assert.Equal(t, expectedPkg.NewVersion, pkg.NewVersion)
			}
		})
	}
}
