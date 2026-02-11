package updater

import (
	"testing"

	"github.com/scottlz0310/devsync/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNvmUpdater_Name(t *testing.T) {
	t.Parallel()

	n := &NvmUpdater{}
	assert.Equal(t, "nvm", n.Name())
}

func TestNvmUpdater_DisplayName(t *testing.T) {
	t.Parallel()

	n := &NvmUpdater{}
	assert.Equal(t, "nvm (Node.js バージョン管理)", n.DisplayName())
}

func TestNvmUpdater_Configure(t *testing.T) {
	t.Parallel()

	n := &NvmUpdater{}
	err := n.Configure(config.ManagerConfig{"dummy": true})
	assert.NoError(t, err)
}

func TestParseNvmCurrentVersion(t *testing.T) {
	testCases := []struct {
		name        string
		output      string
		want        string
		expectErr   bool
		errContains string
	}{
		{
			name:   "通常のv付きバージョン",
			output: "v20.11.1",
			want:   "20.11.1",
		},
		{
			name:   "追加テキストを含む出力",
			output: "v18.19.0 (Currently using 64-bit executable)",
			want:   "18.19.0",
		},
		{
			name:   "noneは未選択として扱う",
			output: "none",
			want:   "",
		},
		{
			name:   "systemは未選択として扱う",
			output: "system",
			want:   "",
		},
		{
			name:        "不正な形式はエラー",
			output:      "not-a-version",
			expectErr:   true,
			errContains: "バージョン形式を解釈できません",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseNvmCurrentVersion(tc.output)
			if tc.expectErr {
				assert.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseLatestNodeVersion(t *testing.T) {
	testCases := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "空出力",
			output: "",
			want:   "",
		},
		{
			name: "nvm ls-remote 形式",
			output: `
      v18.20.4   (LTS: Hydrogen)
      v20.17.0   (LTS: Iron)
      v22.11.0   (Latest LTS: Jod)
`,
			want: "22.11.0",
		},
		{
			name: "nvm list available 形式（Windows系）",
			output: `
|   CURRENT    |     LTS      |  OLD STABLE  | OLD UNSTABLE |
|    22.11.0   |    20.17.0   |   0.12.18    |   0.11.16    |
`,
			want: "22.11.0",
		},
		{
			name: "iojs行は除外",
			output: `
      iojs-v3.3.1
      v20.10.0
`,
			want: "20.10.0",
		},
		{
			name: "不正な行のみ",
			output: `
      stable
      latest
`,
			want: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := parseLatestNodeVersion(tc.output)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsSemverLess(t *testing.T) {
	testCases := []struct {
		name        string
		left        string
		right       string
		want        bool
		expectErr   bool
		errContains string
	}{
		{
			name:  "左が古い",
			left:  "20.10.0",
			right: "20.11.0",
			want:  true,
		},
		{
			name:  "同一バージョン",
			left:  "20.11.0",
			right: "20.11.0",
			want:  false,
		},
		{
			name:  "左が新しい",
			left:  "22.0.0",
			right: "20.11.0",
			want:  false,
		},
		{
			name:        "不正な形式はエラー",
			left:        "20.11",
			right:       "20.12.0",
			expectErr:   true,
			errContains: "不正な semver 形式",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := isSemverLess(tc.left, tc.right)
			if tc.expectErr {
				assert.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
