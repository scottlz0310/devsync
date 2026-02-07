package repo

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// UpdateOptions は repo update の実行オプションです。
type UpdateOptions struct {
	Prune           bool
	AutoStash       bool
	SubmoduleUpdate bool
	DryRun          bool
}

// UpdateResult は単一リポジトリの更新結果です。
type UpdateResult struct {
	RepoPath        string
	Commands        []string
	SkippedMessages []string
	UpstreamChecked bool
	HasUpstream     bool
}

// Update は単一リポジトリに対して fetch/pull/submodule update を実行します。
func Update(ctx context.Context, repoPath string, opts UpdateOptions) (*UpdateResult, error) {
	cleanPath := filepath.Clean(repoPath)

	result := &UpdateResult{
		RepoPath:    cleanPath,
		HasUpstream: true,
	}

	fetchArgs := buildFetchArgs(opts.Prune)
	result.Commands = append(result.Commands, formatGitCommand(cleanPath, fetchArgs))

	if !opts.DryRun {
		if err := runGitCommand(ctx, cleanPath, fetchArgs...); err != nil {
			return result, fmt.Errorf("fetch に失敗: %w", err)
		}
	}

	hasUpstream := true
	if !opts.DryRun {
		upstream, _, err := getAheadCount(ctx, cleanPath)
		if err != nil {
			return result, fmt.Errorf("upstream 確認に失敗: %w", err)
		}

		result.UpstreamChecked = true
		result.HasUpstream = upstream
		hasUpstream = upstream
	}

	pullArgs := buildPullArgs(opts.AutoStash)
	if opts.DryRun || hasUpstream {
		result.Commands = append(result.Commands, formatGitCommand(cleanPath, pullArgs))

		if !opts.DryRun {
			if err := runGitCommand(ctx, cleanPath, pullArgs...); err != nil {
				return result, fmt.Errorf("pull に失敗: %w", err)
			}
		}
	} else {
		result.SkippedMessages = append(result.SkippedMessages, "upstream が未設定のため pull をスキップしました")
	}

	if opts.SubmoduleUpdate {
		submoduleArgs := buildSubmoduleArgs()
		result.Commands = append(result.Commands, formatGitCommand(cleanPath, submoduleArgs))

		if !opts.DryRun {
			if err := runGitCommand(ctx, cleanPath, submoduleArgs...); err != nil {
				return result, fmt.Errorf("submodule update に失敗: %w", err)
			}
		}
	}

	return result, nil
}

func buildFetchArgs(prune bool) []string {
	args := []string{"fetch", "--all"}
	if prune {
		args = append(args, "--prune")
	}

	return args
}

func buildPullArgs(autoStash bool) []string {
	args := []string{"pull", "--rebase"}
	if autoStash {
		args = append(args, "--autostash")
	}

	return args
}

func buildSubmoduleArgs() []string {
	return []string{"submodule", "update", "--init", "--recursive", "--remote"}
}

func formatGitCommand(repoPath string, args []string) string {
	parts := append([]string{"git", "-C", repoPath}, args...)
	return strings.Join(parts, " ")
}

func runGitCommand(ctx context.Context, repoPath string, args ...string) error {
	commandArgs := append([]string{"-C", repoPath}, args...)
	cmd := exec.CommandContext(ctx, "git", commandArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return err
		}

		return fmt.Errorf("%w: %s", err, message)
	}

	return nil
}
