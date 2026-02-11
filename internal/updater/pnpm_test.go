package updater

import (
	"testing"

	"github.com/scottlz0310/devsync/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestPnpmUpdater_Name(t *testing.T) {
	t.Parallel()

	p := &PnpmUpdater{}
	assert.Equal(t, "pnpm", p.Name())
}

func TestPnpmUpdater_DisplayName(t *testing.T) {
	t.Parallel()

	p := &PnpmUpdater{}
	assert.Equal(t, "pnpm (Node.js グローバルパッケージ)", p.DisplayName())
}

func TestPnpmUpdater_Configure(t *testing.T) {
	t.Parallel()

	p := &PnpmUpdater{}
	err := p.Configure(config.ManagerConfig{"dummy": true})
	assert.NoError(t, err)
}

func TestPnpmUpdater_parseOutdatedJSON(t *testing.T) {
	tests := []struct {
		name        string
		output      []byte
		want        map[string]PackageInfo
		expectErr   bool
		errContains string
	}{
		{
			name:      "空出力",
			output:    nil,
			want:      map[string]PackageInfo{},
			expectErr: false,
		},
		{
			name:        "不正なJSONはエラー",
			output:      []byte("{invalid"),
			expectErr:   true,
			errContains: "JSON の解析に失敗",
		},
		{
			name: "配列形式の出力",
			output: []byte(`[
  {"name":"typescript","current":"5.1.0","latest":"5.2.0"},
  {"packageName":"@scope/pkg","current":"1.0.0","wanted":"1.1.0"}
]`),
			want: map[string]PackageInfo{
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
		{
			name: "オブジェクト形式の出力",
			output: []byte(`{
  "eslint": {"current":"8.0.0","latest":"9.0.0"},
  "pnpm": {"current":"9.0.0","wanted":"9.1.0"}
}`),
			want: map[string]PackageInfo{
				"eslint": {
					Name:           "eslint",
					CurrentVersion: "8.0.0",
					NewVersion:     "9.0.0",
				},
				"pnpm": {
					Name:           "pnpm",
					CurrentVersion: "9.0.0",
					NewVersion:     "9.1.0",
				},
			},
		},
		{
			name: "配列形式で名前が空の要素はスキップ",
			output: []byte(`[
  {"name":"", "packageName":"", "current":"1.0.0", "latest":"2.0.0"}
]`),
			want: map[string]PackageInfo{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &PnpmUpdater{}
			got, err := p.parseOutdatedJSON(tt.output)

			if tt.expectErr {
				assert.Error(t, err)

				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}

				return
			}

			assert.NoError(t, err)
			assert.Len(t, got, len(tt.want))

			gotMap := make(map[string]PackageInfo, len(got))
			for _, pkg := range got {
				gotMap[pkg.Name] = pkg
			}

			for name, wantPkg := range tt.want {
				gotPkg, ok := gotMap[name]
				assert.True(t, ok, "package %q が見つかりません", name)
				assert.Equal(t, wantPkg.CurrentVersion, gotPkg.CurrentVersion)
				assert.Equal(t, wantPkg.NewVersion, gotPkg.NewVersion)
			}
		})
	}
}
