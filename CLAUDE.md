# CLAUDE.md

このファイルはこのリポジトリでコードを扱う際のClaude Code (claude.ai/code) へのガイダンスを提供します。

## コマンド

### ビルドと開発
- `go install` - プロバイダーをビルド
- `go generate` - Terraformサンプルをフォーマットしてドキュメントを生成
- `go mod download` - 依存関係をダウンロード
- `go build -v .` - 詳細出力でビルド

### テスト
- `make testacc` - アクセプタンステストを実行（実際のSendGridリソースとAPIキーが必要）
- `go test -v -cover ./internal/provider/` - カバレッジ付きでプロバイダーテストを実行
- `TF_ACC=1 go test ./... -v -timeout 120m` - 拡張タイムアウトで全アクセプタンステストを実行

### コード品質
- `golangci-lint run` - Goリンターを実行（CIで設定済み）
- `terraform fmt -recursive ./examples/` - Terraformサンプルファイルをフォーマット

## アーキテクチャ

これはTerraform Plugin Frameworkを使用して構築されたSendGrid用のTerraformプロバイダーです。

### コア構造
- `main.go` - エントリーポイント、プラグインフレームワーク経由でプロバイダーを提供
- `internal/provider/` - メインプロバイダー実装
  - `provider.go` - プロバイダー設定、リソースとデータソースの登録
  - `*_resource.go` - リソース実装（teammate、api_key、subuserなど）
  - `*_data_source.go` - データソース実装
  - `*_test.go` - リソース/データソースのアクセプタンステスト
- `flex/` - TerraformとGo間の型変換用ユーティリティ関数
- `examples/` - テストとドキュメント用のTerraform設定例
- `docs/` - 自動生成ドキュメント

### 主要パターン
- リソースは `sendgrid_<resource_name>` の命名パターンに従う
- 各リソースには既存リソースを読み取る対応するデータソースがある
- `SENDGRID_API_KEY` 環境変数またはプロバイダー設定による認証
- `SENDGRID_SUBUSER` 環境変数によるオプションのサブユーザーサポート
- `retryOnRateLimit()` 関数でレート制限リトライロジックを実装
- 事前定義リストを使用したteammateリソースの包括的なスコープ検証

### プロバイダー設定
- API Key: 必須、設定または `SENDGRID_API_KEY` 環境変数で設定可能
- Subuser: オプション、設定または `SENDGRID_SUBUSER` 環境変数で設定可能
- github.com/kenzo0107/sendgrid Go クライアントライブラリを使用

### テスト要件
- アクセプタンステストには `SENDGRID_API_KEY` に実際のSendGrid APIキーが必要
- 一部のテストでは IP関連リソース用に `IP_ADDRESS` シークレットが必要
- テストは実際のSendGridリソースを作成し、コストが発生する可能性がある
