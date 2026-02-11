package updater

import (
	"testing"

	"github.com/scottlz0310/devsync/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestGemUpdater_Name(t *testing.T) {
	g := &GemUpdater{}
	assert.Equal(t, "gem", g.Name())
}

func TestGemUpdater_DisplayName(t *testing.T) {
	g := &GemUpdater{}
	assert.Equal(t, "gem (Ruby Gems)", g.DisplayName())
}

func TestGemUpdater_Configure(t *testing.T) {
	g := &GemUpdater{}
	err := g.Configure(config.ManagerConfig{"dummy": true})
	assert.NoError(t, err)
}

func TestGemUpdater_parseOutdatedOutput(t *testing.T) {
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
			name: "通常形式",
			output: `rake (13.1.0 < 13.2.1)
rubocop (1.65.0 < 1.69.1)
`,
			want: []PackageInfo{
				{Name: "rake", CurrentVersion: "13.1.0", NewVersion: "13.2.1"},
				{Name: "rubocop", CurrentVersion: "1.65.0", NewVersion: "1.69.1"},
			},
		},
		{
			name: "current側が複数候補",
			output: `foo (1.0.0, 1.1.0 < 2.0.0)
`,
			want: []PackageInfo{
				{Name: "foo", CurrentVersion: "1.0.0", NewVersion: "2.0.0"},
			},
		},
		{
			name: "defaultラベル付き",
			output: `bundler (default: 2.5.0 < 2.5.12)
`,
			want: []PackageInfo{
				{Name: "bundler", CurrentVersion: "2.5.0", NewVersion: "2.5.12"},
			},
		},
		{
			name: "不正な行は無視",
			output: `invalid line
pkg (1.0.0)
`,
			want: []PackageInfo{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := &GemUpdater{}
			got := g.parseOutdatedOutput(tc.output)
			assert.Equal(t, tc.want, got)
		})
	}
}
