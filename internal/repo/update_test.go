package repo

import (
	"reflect"
	"testing"
)

func TestBuildFetchArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		prune bool
		want  []string
	}{
		{
			name:  "prune有効",
			prune: true,
			want:  []string{"fetch", "--all", "--prune"},
		},
		{
			name:  "prune無効",
			prune: false,
			want:  []string{"fetch", "--all"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildFetchArgs(tc.prune)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("buildFetchArgs(%v) = %v, want %v", tc.prune, got, tc.want)
			}
		})
	}
}

func TestBuildPullArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		autoStash bool
		want      []string
	}{
		{
			name:      "autoStash有効",
			autoStash: true,
			want:      []string{"pull", "--rebase", "--autostash"},
		},
		{
			name:      "autoStash無効",
			autoStash: false,
			want:      []string{"pull", "--rebase"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildPullArgs(tc.autoStash)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("buildPullArgs(%v) = %v, want %v", tc.autoStash, got, tc.want)
			}
		})
	}
}

func TestBuildSubmoduleArgs(t *testing.T) {
	t.Parallel()

	want := []string{"submodule", "update", "--init", "--recursive", "--remote"}
	got := buildSubmoduleArgs()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSubmoduleArgs() = %v, want %v", got, want)
	}
}

func TestFormatGitCommand(t *testing.T) {
	t.Parallel()

	got := formatGitCommand("/tmp/repo", []string{"fetch", "--all", "--prune"})
	want := "git -C /tmp/repo fetch --all --prune"

	if got != want {
		t.Fatalf("formatGitCommand() = %q, want %q", got, want)
	}
}
