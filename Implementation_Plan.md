以下に、ここまでの議論を踏まえた「一押し構成（Go中心 + 安定並列 + インタラクティブ設定）」での **初版の実装計画書（v0.1）** を提示します。
“毎日使う道具”として運用を崩さずに育てられるよう、**価値の出る順に段階導入**する計画にしています。

---

# 実装計画書（初版 / v0.1）

対象：既存ツール（sysup / Setup-Repository）の統合 + Bitwarden 環境変数注入の自動化を含むクロスプラットフォーム CLI

## 1. 目的とゴール

### 目的

* **開発環境運用の統合と一本化**:
  * 既存のシステム更新ツール [sysup](https://github.com/scottlz0310/sysup)
  * リポジトリ管理ツール [Setup-Repository](https://github.com/scottlz0310/Setup-Repository)
  * `bw-cli` (Bitwarden CLI) を用いた環境変数注入の自動化
  
  上記3つの機能を統合し、単一のCLIツールとして運用・保守・拡張の中心に据える。

* **学習体験と技術スタックの刷新**:
  * これまで主力であったPythonによるCLI開発から離れ、**Go**（またはRust）などの静的型付けコンパイル言語を採用する。
  * これにより、配布性・並列処理の安定性を向上させるとともに、新しい言語パラダイムごとの設計・実装パターンを習得する「学習の場」としても活用する。

* **運用体験の向上**:
  * ネットワークアクセスを伴う処理の **安定した並列実行**（高速化と失敗率の低減）。
  * 設定体験を「GUI感覚（ウィザード形式）」に近づけ、継続利用の摩擦を下げる。

### v0.1 ゴール（MVP）

* 単一バイナリで動作するGo CLIが提供され、最低限の日次運用が置き換え可能
* 既存ツール（sysup/Setup-Repository）の機能が移植され、統合コマンドから呼び出せる
* 安定並列（上限・timeout・cancel・集計）が確立し、更新処理が破綻しない
* `config init`（ウィザード）で設定ファイルを作成できる
* `repo update` / `sys update` がそれぞれ動く
* `doctor` で依存コマンド（bwコマンド含む）や基本疎通を確認できる

---

## 2. 推奨技術スタック（v0.1）

* 言語：Go
* CLIフレームワーク：Cobra
* 設定：YAML（人間が読み書きしやすい）＋構造体バインド

  * 読み書き：自前（シンプルなmarshal/unmarshal）または Viper（採用は任意）
* インタラクティブ設定（初期）：survey（initウィザード）
* インタラクティブ設定（次段）：Bubble Tea（TUI編集画面）
* 並列制御：errgroup + semaphore（もしくはワーカープール）
* HTTP：net/http（Transport明示）＋ Context
* ログ：標準log/slog で開始（必要なら zap に移行）
* テスト：標準testing + 必要に応じて testify
* リリース：GoReleaser（Windows/Linux向けバイナリを生成）

---

## 3. CLIのコマンド設計（v0.1）

### 基本コマンド

* `tool update`

  * 内部で repo/sys の双方（または設定に応じて片方）を実行
  * `--jobs N`（並列数）
  * `--timeout 5m`（全体タイムアウト）
  * `--dry-run`（計画のみ）
  * `--yes`（確認なし）
  * `--fail-fast`（任意、デフォルトは集計）

### Repo系

* `tool repo update`
* `tool repo list`（対象一覧表示）
* `tool repo add/remove`（v0.2以降でも可）

### System系

* `tool sys update`
* `tool sys status`（v0.2以降でも可）

### 設定系

* `tool config init`（survey：カーソル選択→確定のウィザード）
* `tool config show`（現在の設定表示）
* `tool config validate`（スキーマ + 実行環境チェック）

### 診断系

* `tool doctor`（validate + 依存コマンド検出 + 疎通テスト）

### 将来（v0.2〜）

* `tool env run --profile X -- <cmd...>`（Bitwarden注入で任意コマンド実行）
* `tool config edit`（Bubble TeaのTUI編集）

---

## 4. 設定スキーマ（初版案）

YAML例（雰囲気）

```yaml
version: 1

concurrency:
  jobs: 6
  timeout: "5m"
  per_request_timeout: "20s"

repo:
  enabled: true
  roots:
    - path: "~/src"
      include:
        - "repoA"
        - "repoB"
      exclude:
        - "legacy-repo"
  actions:
    git_pull: true
    submodule_update: true

sys:
  enabled: true
  managers:
    - kind: "brew"      # macが増えた場合に拡張
    - kind: "apt"
    - kind: "winget"
  options:
    upgrade: true
    cleanup: false

secrets:
  enabled: false
  provider: "bitwarden"
  profiles:
    default:
      mappings:
        - env: "GITHUB_TOKEN"
          item: "github"
          field: "token"
```

**設計方針**

* 設定ファイルには「秘密値そのもの」を保存しない

  * 保存するのは参照情報（item/field/env名）のみ
* OS差分は `sys.managers` で吸収する（複数登録可）
* 並列とtimeoutは設定の中核。コマンドライン引数で上書き可能

---

## 5. バックエンド設計（安定並列の骨格）

### 実行モデル

* すべての処理（repo更新、sys更新、外部コマンド）を **Job** として扱う
* Job実行は以下を必須化

  * 上限並列（jobs）
  * Context cancel（Ctrl+Cで停止）
  * Timeout（全体・個別）
  * 結果集計（success/fail/skip）

### “並列安定性”の必須要件（実装規約）

* 無制限並列は禁止（semaphore/worker pool）
* リトライは条件付き + 指数バックオフ + ジッタ（v0.2でも可）
* レート制限（429 / Retry-After）は尊重（v0.2でも可）
* デフォルトは **fail-fastしない**（集計して最後にサマリ）

### 出力（毎日運用向け）

* 実行開始時：Plan表示（dry-runがあると理想）
* 実行中：進捗（最低限は “N/M 実行中”）
* 実行後：成功/失敗/スキップのサマリ（終了コードで失敗を返す）

---

## 6. インタラクティブ設定（v0.1の範囲）

### v0.1：`config init`（survey）

**狙い：最短で「カーソル選択→確定」を実現し、初期導入を完了させる。**

質問フロー案：

1. repo更新を有効にするか（Yes/No）
2. repoルートディレクトリ候補を提示（検出 + 選択）
3. 対象リポジトリを複数選択（MultiSelect）
4. sys更新を有効にするか（Yes/No）
5. 利用可能なパッケージマネージャを検出して選択
6. 並列数 jobs を選択（推奨値を提示）
7. タイムアウトを選択（推奨値を提示）
8. 生成される設定のプレビュー
9. 保存先を確認して保存

### v0.2：`config edit`（Bubble Tea）

* 設定を “画面” で編集（リスト＋詳細ペイン＋差分レビュー）
* 保存前に差分プレビュー
* validate/doctorへの導線を同画面に持たせる

---

## 7. 実装ステップ（マイルストーン）

### Milestone 0：プロジェクト骨格（0.5日）

* Go module作成
* Cobra導入、`tool --help` でコマンドツリーが出る
* 設定ロード（パス解決、存在しなければ案内）

成果物：

* `tool version`
* `tool config show`（空でも良い）

---

### Milestone 1：実行エンジン（並列・集計）確立（1〜2日）

* Jobインターフェース定義
* semaphore付き実行器
* context cancel、全体timeout、結果集計

成果物：

* ダミーJobで並列実行できる
* サマリが出る
* Ctrl+Cで停止できる

---

### Milestone 2：repo update 実装（1〜2日）

* repo対象解決（roots/include/exclude）
* `git pull` / `submodule update`（必要分）をJob化
* `--dry-run`（plan表示）

成果物：

* `tool repo update` が日次運用に耐える

---

### Milestone 3：sys update 実装（1〜2日）

* manager検出（apt/winget等）と実行
* 実行コマンドはJob化し、並列枠に載せる
* エラーを集計し、最後にまとめる

成果物：

* `tool sys update` が日次運用に耐える

---

### Milestone 4：config init（survey）と validate/doctor（1〜2日）

* `tool config init` ウィザード
* 設定の原子的保存（temp → rename）
* `config validate`（構造チェック＋依存コマンド検出）
* `doctor`（validate + 疎通）

成果物：

* 初期導入がCLI単体で完結する

---

### Milestone 5：配布（0.5〜1日）

* GoReleaserでWindows/Linuxバイナリ作成
* 署名やハッシュ（必要なら）
* README（導入・設定・運用例）

成果物：

* “インストールして使える”状態

---

## 8. 非機能要件（v0.1の品質ライン）

* 安定性：並列数制限 + timeout + cancel は必須
* 可観測性：エラーの要約、失敗ジョブの再実行が容易なログ形式
* 安全性：dry-run / plan表示を優先。破壊的操作は慎重に
* 互換性：Windows/Linuxで同一コマンド体系
* 将来拡張：Bitwarden注入やTUI編集を後付けできる分離設計

---

## 9. v0.2以降の拡張計画（見通し）

* `env run`：Bitwarden注入による任意コマンド実行（bw CLI連携）
* `config edit`：Bubble TeaでのTUI編集
* HTTP処理に対するレート制限対応（429/Retry-After）
* キャッシュ（ネットワーク負荷低減）
* 部分実行（`--only repo` / `--only sys`）
* ジョブ依存関係（ステージング：repo更新 → lock更新 など）

---

# 推奨：この計画の“成功条件”

1. **先に実行エンジン（並列・集計・cancel）を固める**
2. repo/sysは “外部コマンドをJob化して載せる” を徹底
3. 設定は `init → validate → doctor` を先に揃える（運用が安定する）

---

必要なら次に、ここから直結する形で

* **ディレクトリ構成案（cmd/internal/pkg）**
* **Job実行器の雛形（Goコード）**
* **config init の質問フロー詳細（surveyの設計）**
  を、v0.1向けにまとめて出します。
