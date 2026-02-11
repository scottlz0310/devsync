package updater

import (
	"testing"

	"github.com/scottlz0310/devsync/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestFlatpakUpdater_Name(t *testing.T) {
	f := &FlatpakUpdater{}
	assert.Equal(t, "flatpak", f.Name())
}

func TestFlatpakUpdater_DisplayName(t *testing.T) {
	f := &FlatpakUpdater{}
	assert.Equal(t, "Flatpak", f.DisplayName())
}

func TestFlatpakUpdater_Configure(t *testing.T) {
	testCases := []struct {
		name     string
		cfg      config.ManagerConfig
		wantUser bool
		describe string
	}{
		{
			name:     "nilの設定",
			cfg:      nil,
			wantUser: false,
			describe: "nil設定の場合はデフォルト値を維持",
		},
		{
			name:     "新キーuse_user=true",
			cfg:      config.ManagerConfig{"use_user": true},
			wantUser: true,
			describe: "新キーuse_userが有効化される",
		},
		{
			name:     "旧キーuser=true",
			cfg:      config.ManagerConfig{"user": true},
			wantUser: true,
			describe: "旧キーuserも後方互換で受け付ける",
		},
		{
			name:     "新旧キーが競合する場合はuse_userを優先",
			cfg:      config.ManagerConfig{"use_user": false, "user": true},
			wantUser: false,
			describe: "use_userが優先される",
		},
		{
			name:     "不正な型の値は無視",
			cfg:      config.ManagerConfig{"use_user": "true"},
			wantUser: false,
			describe: "bool以外は適用しない",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := &FlatpakUpdater{useUser: false}
			err := f.Configure(tc.cfg)

			assert.NoError(t, err)
			assert.Equal(t, tc.wantUser, f.useUser, tc.describe)
		})
	}
}

func TestFlatpakUpdater_buildCommandArgs(t *testing.T) {
	testCases := []struct {
		name    string
		useUser bool
		args    []string
		want    []string
	}{
		{
			name:    "ユーザー更新無効",
			useUser: false,
			args:    []string{"update", "-y"},
			want:    []string{"update", "-y"},
		},
		{
			name:    "ユーザー更新有効",
			useUser: true,
			args:    []string{"remote-ls", "--updates"},
			want:    []string{"--user", "remote-ls", "--updates"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := &FlatpakUpdater{useUser: tc.useUser}
			got := f.buildCommandArgs(tc.args...)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFlatpakUpdater_parseRemoteLSOutput(t *testing.T) {
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
			name: "ヘッダーのみ",
			output: `Application ID  Version
`,
			want: []PackageInfo{},
		},
		{
			name: "1件のみ",
			output: `org.gnome.Calculator 44.0
`,
			want: []PackageInfo{
				{Name: "org.gnome.Calculator", NewVersion: "44.0"},
			},
		},
		{
			name: "複数件と空行を含む",
			output: `Application ID  Version
org.mozilla.firefox 122.0

org.gnome.TextEditor 45.1
`,
			want: []PackageInfo{
				{Name: "org.mozilla.firefox", NewVersion: "122.0"},
				{Name: "org.gnome.TextEditor", NewVersion: "45.1"},
			},
		},
		{
			name: "バージョン欠落時は名前のみ",
			output: `org.example.Tool
`,
			want: []PackageInfo{
				{Name: "org.example.Tool"},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := &FlatpakUpdater{}
			got := f.parseRemoteLSOutput(tc.output)

			assert.Equal(t, tc.want, got)
		})
	}
}
