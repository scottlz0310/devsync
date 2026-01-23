package secret

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

// Injector はシークレットを注入する機能を提供します。
type Injector struct {
	Items []string
}

// BitwardenItem は `bw get item` のJSON出力の一部に対応する構造体です。
type BitwardenItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Notes string `json:"notes"`
	Login struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"login"`
}

// NewInjector は新しいInjectorを作成します。
func NewInjector(items []string) *Injector {
	return &Injector{Items: items}
}

// Inject は設定されたアイテムをBitwardenから取得し、環境変数に注入します。
// メモ欄に含まれる `env:VAR_NAME` という記述を環境変数名として使用します。
func (i *Injector) Inject() error {
	if len(i.Items) == 0 {
		return nil
	}

	// bwコマンドの存在確認
	if _, err := exec.LookPath("bw"); err != nil {
		return fmt.Errorf("bw command not found. please install Bitwarden CLI")
	}

	// セッションチェック (簡易的)
	if os.Getenv("BW_SESSION") == "" {
		fmt.Println("Bitwarden session token not found (BW_SESSION).")
		fmt.Println("Please run 'bw login' or 'bw unlock' and export BW_SESSION.")
		// ここで対話的に unlock する実装も考えられるが、まずはエラーにするかWarningにする
		// unlockはセキュリティクリティカルなので、CLIでの入力を求めるなら go-password とかが必要
		return fmt.Errorf("BW_SESSION environment variable is not set")
	}

	fmt.Println("🔒 Bitwarden からシークレットを取得中...")

	for _, itemID := range i.Items {
		item, err := i.getItem(itemID)
		if err != nil {
			return fmt.Errorf("item '%s' の取得失敗: %w", itemID, err)
		}

		envName := i.extractEnvName(item.Notes)
		if envName == "" {
			fmt.Printf("⚠️ Item '%s' (%s) のメモに 'env:NAME' の指定が見つかりません。スキップします。\n", itemID, item.Name)
			continue
		}

		if item.Login.Password == "" {
			fmt.Printf("⚠️ Item '%s' (%s) のパスワードが空です。スキップします。\n", itemID, item.Name)
			continue
		}

		// 環境変数にセット
		if err := os.Setenv(envName, item.Login.Password); err != nil {
			return fmt.Errorf("failed to set env var %s: %w", envName, err)
		}
		fmt.Printf("🔑 環境変数を注入しました: %s\n", envName)
	}

	return nil
}

func (i *Injector) getItem(id string) (*BitwardenItem, error) {
	// bw get item <id> --raw
	cmd := exec.Command("bw", "get", "item", id, "--raw")
	// BW_SESSION は親プロセスから継承される
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var item BitwardenItem
	if err := json.Unmarshal(output, &item); err != nil {
		return nil, err
	}

	return &item, nil
}

// extractEnvName はメモ欄から `env:VAR_NAME` 形式の記述を探して返します。
func (i *Injector) extractEnvName(notes string) string {
	// 正規表現: env: に続く 英大文字・数字・アンダースコア
	re := regexp.MustCompile(`env:([A-Z0-9_]+)`)
	matches := re.FindStringSubmatch(notes)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
