package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/scottlz0310/devsync/internal/config"
	repomgr "github.com/scottlz0310/devsync/internal/repo"
	"github.com/spf13/cobra"
)

var repoRootOverride string

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "リポジトリ管理",
	Long:  `管理対象リポジトリの検出・状態確認・更新を行います。`,
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "管理下リポジトリの一覧を表示します",
	Long: `設定された root 配下の Git リポジトリを検出し、
状態（クリーン/ダーティ/未プッシュ/追跡なし）を表示します。`,
	RunE: runRepoList,
}

func init() {
	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoListCmd)

	repoListCmd.Flags().StringVar(&repoRootOverride, "root", "", "スキャン対象のルートディレクトリ（指定時は設定を上書き）")
}

func runRepoList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  設定ファイルの読み込みに失敗（デフォルト設定を使用）: %v\n", err)

		cfg = config.Default()
	}

	root := cfg.Repo.Root
	if cmd.Flags().Changed("root") {
		root = repoRootOverride
	}

	timeout := 10 * time.Minute
	if parsed, parseErr := time.ParseDuration(cfg.Control.Timeout); parseErr == nil {
		timeout = parsed
	}

	baseCtx := cmd.Context()
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	ctx, cancel := context.WithTimeout(baseCtx, timeout)
	defer cancel()

	repos, err := repomgr.List(ctx, root)
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		fmt.Printf("📝 リポジトリが見つかりませんでした: %s\n", root)
		return nil
	}

	fmt.Printf("📦 管理下リポジトリ一覧 (%d件)\n\n", len(repos))

	if err := printRepoTable(repos); err != nil {
		return fmt.Errorf("一覧表示に失敗: %w", err)
	}

	return nil
}

func printRepoTable(repos []repomgr.Info) error {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)

	if _, err := fmt.Fprintln(writer, "名前\t状態\tAhead\tパス"); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(writer, "----\t----\t-----\t----"); err != nil {
		return err
	}

	for _, repo := range repos {
		ahead := "-"
		if repo.HasUpstream {
			ahead = strconv.Itoa(repo.Ahead)
		}

		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", repo.Name, repomgr.StatusLabel(repo.Status), ahead, repo.Path); err != nil {
			return err
		}
	}

	return writer.Flush()
}
