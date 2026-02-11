package updater

import (
	"testing"

	"github.com/scottlz0310/devsync/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestUVUpdater_Name(t *testing.T) {
	u := &UVUpdater{}
	assert.Equal(t, "uv", u.Name())
}

func TestUVUpdater_DisplayName(t *testing.T) {
	u := &UVUpdater{}
	assert.Equal(t, "uv tool (Python CLI ツール)", u.DisplayName())
}

func TestUVUpdater_Configure(t *testing.T) {
	u := &UVUpdater{}
	err := u.Configure(config.ManagerConfig{"dummy": true})
	assert.NoError(t, err)
}

func TestUVUpdater_parseToolListOutput(t *testing.T) {
	testCases := []struct {
		name   string
		output string
		want   []PackageInfo
	}{
		{
			name:   "空出力",
			output: "",
			want:   []PackageInfo{},
		},
		{
			name:   "未インストールメッセージは対象外",
			output: "No tools installed\n",
			want:   []PackageInfo{},
		},
		{
			name: "通常の一覧",
			output: `ruff v0.6.2
- ruff
httpie v3.2.2
- http
`,
			want: []PackageInfo{
				{Name: "ruff", CurrentVersion: "0.6.2"},
				{Name: "httpie", CurrentVersion: "3.2.2"},
			},
		},
		{
			name: "バージョンなし行も名前だけ保持",
			output: `custom-tool
`,
			want: []PackageInfo{
				{Name: "custom-tool"},
			},
		},
		{
			name: "コロンや括弧を含む形式",
			output: `black v24.10.0:
pkg 1.2.3 (/tmp/path)
`,
			want: []PackageInfo{
				{Name: "black", CurrentVersion: "24.10.0"},
				{Name: "pkg", CurrentVersion: "1.2.3"},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			u := &UVUpdater{}
			got := u.parseToolListOutput(tc.output)
			assert.Equal(t, tc.want, got)
		})
	}
}
