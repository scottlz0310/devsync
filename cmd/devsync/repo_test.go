package main

import "testing"

func TestResolveRepoJobs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		configJobs int
		flagJobs   int
		want       int
	}{
		{
			name:       "フラグ優先",
			configJobs: 8,
			flagJobs:   3,
			want:       3,
		},
		{
			name:       "フラグ未指定なら設定値",
			configJobs: 6,
			flagJobs:   0,
			want:       6,
		},
		{
			name:       "設定が不正なら1",
			configJobs: 0,
			flagJobs:   0,
			want:       1,
		},
		{
			name:       "負数フラグは設定値にフォールバック",
			configJobs: 5,
			flagJobs:   -1,
			want:       5,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolveRepoJobs(tc.configJobs, tc.flagJobs)
			if got != tc.want {
				t.Fatalf("resolveRepoJobs(%d, %d) = %d, want %d", tc.configJobs, tc.flagJobs, got, tc.want)
			}
		})
	}
}
