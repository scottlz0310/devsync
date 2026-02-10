package updater

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipxUpdater_parsePipxListJSON(t *testing.T) {
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
			name: "インストール済みパッケージあり",
			output: []byte(`{
  "venvs": {
    "httpie": { "metadata": { "main_package": { "package_version": "3.0.0" } } },
    "black": { "metadata": { "main_package": { "package_version": "23.1.0" } } }
  }
}`),
			expected: map[string]PackageInfo{
				"httpie": {Name: "httpie", CurrentVersion: "3.0.0", NewVersion: ""},
				"black":  {Name: "black", CurrentVersion: "23.1.0", NewVersion: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PipxUpdater{}
			got := p.parsePipxListJSON(tt.output)

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
