package updater

import (
	"testing"
)

func TestScoopParseStatusOutput(t *testing.T) {
	s := &ScoopUpdater{}

	tests := []struct {
		name     string
		input    string
		expected []PackageInfo
	}{
		{
			name: "複数パッケージの更新あり",
			input: `Scoop is up to date.
Name              Installed Version   Latest Version   Missing Dependencies   Info
----              -----------------   --------------   --------------------   ----
git               2.34.1              2.38.0
nodejs            16.13.0             18.9.0
python            3.9.7               3.10.4
`,
			expected: []PackageInfo{
				{Name: "git", CurrentVersion: "2.34.1", NewVersion: "2.38.0"},
				{Name: "nodejs", CurrentVersion: "16.13.0", NewVersion: "18.9.0"},
				{Name: "python", CurrentVersion: "3.9.7", NewVersion: "3.10.4"},
			},
		},
		{
			name: "更新なし",
			input: `Scoop is up to date.
Everything is ok!
`,
			expected: nil,
		},
		{
			name:     "空出力",
			input:    "",
			expected: nil,
		},
		{
			name: "1件のみ",
			input: `Name              Installed Version   Latest Version
----              -----------------   --------------
git               2.34.1              2.38.0
`,
			expected: []PackageInfo{
				{Name: "git", CurrentVersion: "2.34.1", NewVersion: "2.38.0"},
			},
		},
		{
			name: "Missing Dependencies と Info カラムあり",
			input: `Name              Installed Version   Latest Version   Missing Dependencies   Info
----              -----------------   --------------   --------------------   ----
7zip              21.07               22.01                                   Version changed
git               2.34.1              2.38.0
`,
			expected: []PackageInfo{
				{Name: "7zip", CurrentVersion: "21.07", NewVersion: "22.01"},
				{Name: "git", CurrentVersion: "2.34.1", NewVersion: "2.38.0"},
			},
		},
		{
			name: "WARN メッセージを含む出力",
			input: `WARN  Scoop bucket(s) out of date. Run 'scoop update' to get the latest changes.
Name              Installed Version   Latest Version
----              -----------------   --------------
git               2.34.1              2.38.0
`,
			expected: []PackageInfo{
				{Name: "git", CurrentVersion: "2.34.1", NewVersion: "2.38.0"},
			},
		},
		{
			name: "ヘッダーのみ・データなし",
			input: `Name              Installed Version   Latest Version
----              -----------------   --------------
`,
			expected: []PackageInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.parseStatusOutput(tt.input)

			if len(got) != len(tt.expected) {
				t.Fatalf("パッケージ数が不一致: got %d, want %d\ngot: %+v", len(got), len(tt.expected), got)
			}

			for i, pkg := range got {
				exp := tt.expected[i]
				if pkg.Name != exp.Name {
					t.Errorf("[%d] Name: got %q, want %q", i, pkg.Name, exp.Name)
				}

				if pkg.CurrentVersion != exp.CurrentVersion {
					t.Errorf("[%d] CurrentVersion: got %q, want %q", i, pkg.CurrentVersion, exp.CurrentVersion)
				}

				if pkg.NewVersion != exp.NewVersion {
					t.Errorf("[%d] NewVersion: got %q, want %q", i, pkg.NewVersion, exp.NewVersion)
				}
			}
		})
	}
}

func TestIsScoopSeparator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"ダッシュのみ", "----------", true},
		{"ダッシュとスペース", "----  -----  ----", true},
		{"文字を含む", "---a---", false},
		{"スペースのみ", "     ", false},
		{"空文字列", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isScoopSeparator(tt.input)
			if got != tt.expected {
				t.Errorf("isScoopSeparator(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDetectScoopColumnPositions(t *testing.T) {
	tests := []struct {
		name        string
		header      string
		minExpected int
	}{
		{
			name:        "通常のヘッダー",
			header:      "Name              Installed Version   Latest Version",
			minExpected: 3,
		},
		{
			name:        "5カラム",
			header:      "Name              Installed Version   Latest Version   Missing Dependencies   Info",
			minExpected: 5,
		},
		{
			name:        "1カラム",
			header:      "Name",
			minExpected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectScoopColumnPositions(tt.header)
			if len(got) < tt.minExpected {
				t.Errorf("カラム位置数: got %d, want >= %d (positions: %v)", len(got), tt.minExpected, got)
			}
		})
	}
}
