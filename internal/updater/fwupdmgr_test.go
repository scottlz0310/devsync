package updater

import (
	"errors"
	"testing"

	"github.com/scottlz0310/devsync/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestFwupdmgrUpdater_Name(t *testing.T) {
	f := &FwupdmgrUpdater{}
	assert.Equal(t, "fwupdmgr", f.Name())
}

func TestFwupdmgrUpdater_DisplayName(t *testing.T) {
	f := &FwupdmgrUpdater{}
	assert.Equal(t, "fwupdmgr (Linux Firmware)", f.DisplayName())
}

func TestFwupdmgrUpdater_Configure(t *testing.T) {
	f := &FwupdmgrUpdater{}
	err := f.Configure(config.ManagerConfig{"dummy": true})
	assert.NoError(t, err)
}

func TestFwupdmgrUpdater_parseGetUpdatesJSON(t *testing.T) {
	testCases := []struct {
		name   string
		output []byte
		want   []PackageInfo
	}{
		{
			name:   "空出力",
			output: nil,
			want:   nil,
		},
		{
			name:   "不正なJSON",
			output: []byte("{invalid"),
			want:   nil,
		},
		{
			name:   "devicesキーがない",
			output: []byte(`{"status":"ok"}`),
			want:   nil,
		},
		{
			name: "更新対象1件",
			output: []byte(`{
  "Devices": [
    {
      "Name": "USB-C Dock",
      "CurrentVersion": "1.0.0",
      "Releases": [
        {"Version": "1.1.0"}
      ]
    }
  ]
}`),
			want: []PackageInfo{
				{
					Name:           "USB-C Dock",
					CurrentVersion: "1.0.0",
					NewVersion:     "1.1.0",
				},
			},
		},
		{
			name: "無効データはスキップ",
			output: []byte(`{
  "devices": [
    {"name":"NoRelease", "releases":[]},
    {"releases":[{"version":"2.0.0"}]},
    {"name":"NoVersion", "releases":[{}]}
  ]
}`),
			want: []PackageInfo{},
		},
		{
			name: "nameが空でもguidにフォールバック",
			output: []byte(`{
  "devices": [
    {
      "guid": "abcd-efgh",
      "version": "1.0",
      "releases": [{"version":"1.1"}]
    }
  ]
}`),
			want: []PackageInfo{
				{
					Name:           "abcd-efgh",
					CurrentVersion: "1.0",
					NewVersion:     "1.1",
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := &FwupdmgrUpdater{}
			got := f.parseGetUpdatesJSON(tc.output)

			if tc.want == nil {
				assert.Nil(t, got)
				return
			}

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsNoFwupdmgrUpdatesMessage(t *testing.T) {
	testCases := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "no updatable devices",
			output: "No updatable devices",
			want:   true,
		},
		{
			name:   "no updates available",
			output: "There are no updates available",
			want:   true,
		},
		{
			name:   "no upgrades for",
			output: "No upgrades for device",
			want:   true,
		},
		{
			name:   "それ以外はfalse",
			output: "device update failed",
			want:   false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isNoFwupdmgrUpdatesMessage(tc.output)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildFwupdmgrOutputErr(t *testing.T) {
	baseErr := errors.New("base error")

	t.Run("出力なしは元エラーを返す", func(t *testing.T) {
		t.Parallel()

		got := buildFwupdmgrOutputErr(baseErr, nil)
		assert.ErrorIs(t, got, baseErr)
		assert.Equal(t, "base error", got.Error())
	})

	t.Run("出力ありはメッセージを連結", func(t *testing.T) {
		t.Parallel()

		got := buildFwupdmgrOutputErr(baseErr, []byte("details"))
		assert.ErrorIs(t, got, baseErr)
		assert.Contains(t, got.Error(), "details")
	})
}
