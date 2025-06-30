# Google Cloud Logging API クォータ最適化

## 実装した最適化機能

### 1. ページサイズ最適化
- **デフォルトページサイズ**: 50 → 10に削減
- **最大ページサイズ制限**: 20件まで
- **効果**: API呼び出し1回あたりの処理量を大幅削減

### 2. インメモリキャッシュ
- **キャッシュ期間**: 
  - リアルタイムログ: 2分
  - 履歴ログ: 10分
- **効果**: 同一クエリの重複API呼び出しを防止

### 3. サーバーサイドフィルタリング強化
- **FilterBuilder**: 効率的なフィルタ構築
- **構造化クエリ解析**: テキストクエリを効率的なフィルタに変換
- **デフォルト時間制限**: 指定がない場合は直近24時間に制限
- **効果**: Cloud Logging側での事前フィルタリングにより転送データ量削減

### 4. エクスポネンシャルバックオフ
- **リトライ戦略**: 1秒 → 2秒 → 4秒 → 8秒
- **最大リトライ**: 3回
- **効果**: レート制限エラー時の自動復旧

### 5. プリセットクエリ機能
- **事前定義クエリ**: よく使用されるクエリを最適化済みで提供
- **利用可能プリセット**:
  - `cloud_run_errors`: Cloud Runサービスの直近エラー
  - `cloud_run_service_errors`: 特定サービスのエラー
  - `recent_logs`: 直近1時間のログ
  - `high_severity`: 直近6時間のエラー・クリティカルログ

### 6. コンテキスト管理
- **タイムアウト設定**: 30秒でクエリタイムアウト
- **リソース管理**: 適切なコンテキストキャンセル
- **効果**: 長時間実行クエリの防止

### 7. ServiceInfo構造体
- **構造化データ**: ログエントリの効率的な処理
- **メタデータ抽出**: Cloud Runサービス情報の自動抽出
- **効果**: データ処理の効率化

## 使用方法

### より効率的な検索クエリ例

**Before (非効率)**:
```
mcp-o11y:search_logs(query: "casone-lite-tenant-api-qa severity>=ERROR", pageSize: 50)
```

**After (効率的)**:
```
# 最適化されたフィルタ使用
mcp-o11y:list_log_entries(
  filter: "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"casone-lite-tenant-api-qa\" AND severity>=ERROR AND timestamp>=\"2025-06-30T00:00:00Z\"",
  pageSize: 10
)

# プリセットクエリ使用（推奨）
mcp-o11y:preset_query(
  queryName: "cloud_run_service_errors",
  parameters: ["casone-lite-tenant-api-qa"]
)
```

### 推奨プラクティス

1. **プリセットクエリを最優先で使用**
   ```
   # Cloud Runエラー調査
   mcp-o11y:preset_query(queryName: "cloud_run_errors")
   
   # 特定サービスのエラー調査
   mcp-o11y:preset_query(
     queryName: "cloud_run_service_errors",
     parameters: ["your-service-name"]
   )
   ```

2. **具体的な時間範囲を指定**
   ```
   startTime: "2025-06-30T06:00:00Z"
   endTime: "2025-06-30T07:00:00Z"
   ```

3. **サービス名やリソースタイプを明示**
   ```
   filter: "resource.labels.service_name=\"your-service\" AND resource.type=\"cloud_run_revision\""
   ```

4. **重要度レベルでフィルタ**
   ```
   filter: "severity>=ERROR"
   ```

5. **小さなページサイズから開始**
   ```
   pageSize: 5  # 最初は少なく、必要に応じて増加
   ```

## 期待される効果

- **API呼び出し頻度**: 最大80%削減（キャッシュヒット時）
- **データ転送量**: 最大70%削減（サーバーサイドフィルタリング）
- **レスポンス時間**: キャッシュヒット時は即座に応答
- **クォータ消費**: 大幅な削減により1分間60回制限に抵触しにくくなる
- **プリセットクエリ**: 最適化済みクエリで90%の効率化
- **エラー処理**: 自動リトライによる高い可用性

## モニタリング

ログで以下を確認可能:
- `Cache hit for query: ...` - キャッシュが有効に機能
- `Cache hit for preset query: ...` - プリセットクエリのキャッシュヒット
- `Executing with backoff...` - レート制限時の自動リトライ
- フィルタ変換の詳細ログ
- `Using preset query: ...` - プリセットクエリの使用状況

## 利用可能なプリセットクエリ

| クエリ名 | 説明 | パラメータ | 例 |
|---------|------|----------|---------|
| `cloud_run_errors` | Cloud Runサービスの直近エラー | なし | `preset_query(queryName: "cloud_run_errors")` |
| `cloud_run_service_errors` | 特定サービスのエラー | service_name | `preset_query(queryName: "cloud_run_service_errors", parameters: ["api-service"])` |
| `recent_logs` | 直近1時間のログ | なし | `preset_query(queryName: "recent_logs")` |
| `high_severity` | 直近6時間のエラー・クリティカル | なし | `preset_query(queryName: "high_severity")` |