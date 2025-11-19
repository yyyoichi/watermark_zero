# Database Integration for Watermark Optimization

SQLite3データベースを使用した実験データの永続化とクエリシステム。

## 特徴

- **Pure Go実装**: `modernc.org/sqlite`を使用し、cgoやシステムライブラリ不要
- **正規化されたスキーマ**: 画像、マーク、パラメータ、結果を効率的に管理
- **柔軟なクエリ**: SQLで自由にデータを分析可能
- **重複排除**: 同じデータの重複保存を自動的に防止

## セットアップ

```bash
cd /workspaces/watermark_zero/exp
go get modernc.org/sqlite
```

## データベーススキーマ

### テーブル構造

- **images**: 画像URL（重複排除）
- **image_sizes**: リサイズされた画像サイズ
- **marks**: オリジナルのウォーターマーク
- **ecc_marks**: エンコード済みウォーターマーク（アルゴリズム別）
- **mark_params**: 埋め込みパラメータ（BlockShape, D1, D2）
- **results**: テスト結果

### ビュー

- **results_detailed**: すべてのテーブルをJOINした詳細ビュー

## 使用方法

### 1. 実験実行（データ収集）

```bash
cd /workspaces/watermark_zero/exp
go run ./cmd/optimize -n 10 -offset 0 -ec-low 1.0 -ec-high 8.0
```

データは `./tmp/optimize/jsons/optimize_results.db` に保存されます。

### 2. データクエリ

#### 統計情報の表示

```bash
go run ./cmd/query -query stats
```

#### 成功率の高いパラメータを取得

```bash
go run ./cmd/query -query best-params -min-success 0.8
```

出力例:
```json
[
  {
    "BlockShapeH": 8,
    "BlockShapeW": 8,
    "D1": 19,
    "D2": 9,
    "TotalTests": 150,
    "Successes": 142,
    "SuccessRate": 0.947,
    "AvgSSIM": 0.9823,
    "AvgAccuracy": 98.5
  }
]
```

#### 画像サイズ別の統計

```bash
go run ./cmd/query -query image-sizes
```

#### EmbedCount範囲別の統計

```bash
go run ./cmd/query -query embed-counts
```

#### 成功した結果のみ取得

```bash
go run ./cmd/query -query successful -min-ssim 0.95
```

#### カスタムSQLクエリ

```bash
go run ./cmd/query -query raw -sql "SELECT d1, d2, AVG(ssim) as avg_ssim FROM results_detailed WHERE success = 1 GROUP BY d1, d2 ORDER BY avg_ssim DESC"
```

## Go APIでの使用

```go
package main

import (
    "exp/internal/db"
    "log"
)

func main() {
    // データベースを開く
    database, err := db.Open("./tmp/optimize/jsons/optimize_results.db")
    if err != nil {
        log.Fatal(err)
    }
    defer database.Close()

    // 成功率の高いパラメータを取得
    params, err := database.GetBestParameters(0.8)
    if err != nil {
        log.Fatal(err)
    }

    for _, p := range params {
        log.Printf("D1=%d D2=%d: Success Rate=%.2f%%, Avg SSIM=%.4f\n",
            p.D1, p.D2, p.SuccessRate*100, p.AvgSSIM)
    }

    // 特定のEmbedCount範囲で検索
    results, err := database.GetResultsByEmbedCount(4.0, 6.0)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Found %d results in EmbedCount range 4.0-6.0\n", len(results))

    // 画像サイズ別の統計
    stats, err := database.GetImageSizeStats()
    if err != nil {
        log.Fatal(err)
    }

    for _, s := range stats {
        log.Printf("%dx%d: Success Rate=%.2f%%, Tests=%d\n",
            s.Width, s.Height, s.SuccessRate*100, s.TotalTests)
    }
}
```

## 便利なSQLクエリ例

### D1/D2ヒートマップ用データ

```sql
SELECT d1, d2, AVG(success) as success_rate, AVG(ssim) as avg_ssim
FROM results_detailed
GROUP BY d1, d2
ORDER BY d1, d2;
```

### 画像サイズとEmbedCountの関係

```sql
SELECT width, height, AVG(embed_count) as avg_embed_count, AVG(success) as success_rate
FROM results_detailed
GROUP BY width, height
ORDER BY avg_embed_count;
```

### 最も高いSSIMを持つパラメータ

```sql
SELECT d1, d2, MAX(ssim) as max_ssim, COUNT(*) as tests
FROM results_detailed
WHERE success = 1
GROUP BY d1, d2
ORDER BY max_ssim DESC
LIMIT 10;
```

### EmbedCount閾値の分析

```sql
SELECT 
    ROUND(embed_count) as embed_count_rounded,
    COUNT(*) as total,
    SUM(CASE WHEN success THEN 1 ELSE 0 END) as successes,
    AVG(ssim) as avg_ssim
FROM results_detailed
GROUP BY embed_count_rounded
ORDER BY embed_count_rounded;
```

## MCP統合（将来）

このデータベースはModel Context Protocol (MCP)サーバーと統合して、AIがデータをクエリできるようになります。

```json
{
  "mcpServers": {
    "watermark-optimizer": {
      "command": "go",
      "args": ["run", "./cmd/mcp-server"],
      "env": {
        "DB_PATH": "./tmp/optimize/jsons/optimize_results.db"
      }
    }
  }
}
```

## データのエクスポート

### JSON形式

```bash
sqlite3 ./tmp/optimize/jsons/optimize_results.db \
  ".mode json" \
  ".output results.json" \
  "SELECT * FROM results_detailed;"
```

### CSV形式

```bash
sqlite3 ./tmp/optimize/jsons/optimize_results.db \
  ".mode csv" \
  ".output results.csv" \
  "SELECT * FROM results_detailed;"
```

## トラブルシューティング

### データベースが見つからない

```bash
# データベースファイルの場所を確認
ls -lh ./tmp/optimize/jsons/optimize_results.db
```

### データベースのリセット

```bash
# データベースファイルを削除して再作成
rm ./tmp/optimize/jsons/optimize_results.db
go run ./cmd/optimize -n 1 -offset 0 -ec-low 1.0 -ec-high 8.0
```

### スキーマの確認

```bash
sqlite3 ./tmp/optimize/jsons/optimize_results.db ".schema"
```
