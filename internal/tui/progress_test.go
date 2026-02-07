package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/scottlz0310/devsync/internal/runner"
)

func TestModelApplyEvent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		events            []runner.Event
		wantState         jobState
		wantLogContains   string
		wantErrorContains string
	}{
		{
			name: "成功終了",
			events: []runner.Event{
				{Type: runner.EventStarted, JobIndex: 0, JobName: "job-1", Timestamp: time.Now()},
				{Type: runner.EventFinished, JobIndex: 0, JobName: "job-1", Status: runner.StatusSuccess, Duration: 200 * time.Millisecond, Timestamp: time.Now()},
			},
			wantState:       jobSuccess,
			wantLogContains: "完了: job-1",
		},
		{
			name: "失敗終了",
			events: []runner.Event{
				{Type: runner.EventStarted, JobIndex: 0, JobName: "job-1", Timestamp: time.Now()},
				{Type: runner.EventFinished, JobIndex: 0, JobName: "job-1", Status: runner.StatusFailed, Err: errors.New("boom"), Duration: 100 * time.Millisecond, Timestamp: time.Now()},
			},
			wantState:         jobFailed,
			wantLogContains:   "失敗: job-1",
			wantErrorContains: "boom",
		},
		{
			name: "スキップ終了",
			events: []runner.Event{
				{Type: runner.EventFinished, JobIndex: 0, JobName: "job-1", Status: runner.StatusSkipped, Err: context.Canceled, Duration: 50 * time.Millisecond, Timestamp: time.Now()},
			},
			wantState:         jobSkipped,
			wantLogContains:   "スキップ: job-1",
			wantErrorContains: "context canceled",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModel("test", []runner.Job{{Name: "job-1"}})
			for _, event := range tc.events {
				event := event
				m.applyEvent(&event)
			}

			if len(m.jobs) != 1 {
				t.Fatalf("job count = %d, want 1", len(m.jobs))
			}

			if m.jobs[0].State != tc.wantState {
				t.Fatalf("state = %s, want %s", m.jobs[0].State, tc.wantState)
			}

			if tc.wantErrorContains != "" && !strings.Contains(m.jobs[0].Err, tc.wantErrorContains) {
				t.Fatalf("err = %q, want contains %q", m.jobs[0].Err, tc.wantErrorContains)
			}

			if tc.wantLogContains != "" && !containsLog(m.logs, tc.wantLogContains) {
				t.Fatalf("logs = %+v, want contains %q", m.logs, tc.wantLogContains)
			}
		})
	}
}

func TestRenderBar(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		percent float64
		want    string
	}{
		{
			name:    "0未満は0扱い",
			percent: -1,
			want:    "[----------]",
		},
		{
			name:    "0.5は半分",
			percent: 0.5,
			want:    "[=====-----]",
		},
		{
			name:    "1超えは最大",
			percent: 2,
			want:    "[==========]",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := renderBar(tc.percent, 10)
			if got != tc.want {
				t.Fatalf("renderBar(%v, 10) = %q, want %q", tc.percent, got, tc.want)
			}
		})
	}
}

func TestProgressPercent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		state jobState
		frame int
		min   float64
		max   float64
	}{
		{
			name:  "待機は0",
			state: jobPending,
			frame: 0,
			min:   0,
			max:   0,
		},
		{
			name:  "実行中は0.2-0.7",
			state: jobRunning,
			frame: 3,
			min:   0.2,
			max:   0.7,
		},
		{
			name:  "完了は1",
			state: jobSuccess,
			frame: 0,
			min:   1,
			max:   1,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := progressPercent(tc.state, tc.frame)
			if got < tc.min || got > tc.max {
				t.Fatalf("progressPercent(%s, %d) = %v, want between %v and %v", tc.state, tc.frame, got, tc.min, tc.max)
			}
		})
	}
}

func containsLog(logs []logEntry, needle string) bool {
	for _, log := range logs {
		if strings.Contains(log.Message, needle) {
			return true
		}
	}

	return false
}
