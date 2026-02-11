package updater

import (
	"testing"

	"github.com/scottlz0310/devsync/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestRustupUpdater_Name(t *testing.T) {
	r := &RustupUpdater{}
	assert.Equal(t, "rustup", r.Name())
}

func TestRustupUpdater_DisplayName(t *testing.T) {
	r := &RustupUpdater{}
	assert.Equal(t, "rustup (Rust ツールチェーン)", r.DisplayName())
}

func TestRustupUpdater_Configure(t *testing.T) {
	r := &RustupUpdater{}
	err := r.Configure(config.ManagerConfig{"dummy": true})
	assert.NoError(t, err)
}

func TestRustupUpdater_parseCheckOutput(t *testing.T) {
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
			name: "更新なし",
			output: `stable-x86_64-unknown-linux-gnu - Up to date : 1.81.0
rustup - Up to date : 1.28.1
`,
			want: []PackageInfo{},
		},
		{
			name: "更新あり",
			output: `stable-x86_64-unknown-linux-gnu - Update available : 1.81.0 -> 1.82.0
rustup - Update available : 1.28.1 -> 1.29.0
`,
			want: []PackageInfo{
				{
					Name:           "stable-x86_64-unknown-linux-gnu",
					CurrentVersion: "1.81.0",
					NewVersion:     "1.82.0",
				},
				{
					Name:           "rustup",
					CurrentVersion: "1.28.1",
					NewVersion:     "1.29.0",
				},
			},
		},
		{
			name: "付加情報付きの更新行",
			output: `nightly-x86_64-unknown-linux-gnu - Update available : 2025-01-01 (abcd123) -> 2025-01-08 (efgh456)
`,
			want: []PackageInfo{
				{
					Name:           "nightly-x86_64-unknown-linux-gnu",
					CurrentVersion: "2025-01-01",
					NewVersion:     "2025-01-08",
				},
			},
		},
		{
			name: "不正な行は無視",
			output: `broken line
toolchain - Update available
`,
			want: []PackageInfo{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &RustupUpdater{}
			got := r.parseCheckOutput(tc.output)
			assert.Equal(t, tc.want, got)
		})
	}
}
