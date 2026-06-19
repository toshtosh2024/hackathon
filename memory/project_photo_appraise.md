---
name: project-photo-appraise
description: SellScreen に Gemini Vision + Google Search grounding を使ったカメラAI査定機能を実装した記録
metadata:
  type: project
---

## 実装した機能: カメラAI査定（写真→相場→価格提案）

**Why:** 出品画面でユーザーが商品写真を撮ると、Gemini APIで商品を識別しウェブ相場を調べ、状態を加味した価格提案をするという機能のリクエスト。

**How to apply:** 今後この機能を拡張・デバッグする際の出発点として参照。

---

### 追加・変更ファイル

| ファイル | 種別 | 内容 |
|---|---|---|
| `backend/cmd/api/handlers_ai_gemini.go` | 新規 | Gemini Vision・Search grounding ハンドラー |
| `backend/cmd/api/main.go` | 変更（1行） | `POST /api/ai/photo-appraise` ルートを追加 |
| `frontend/src/PhotoAppraiser.tsx` | 新規 | カメラ撮影・AI査定UIコンポーネント |
| `frontend/src/SellScreen.tsx` | 変更 | PhotoAppraiser を統合（既存フォームは変更なし） |

---

### バックエンドの処理フロー（2ステップ）

1. **Step1 – Gemini Vision（検索なし・JSON出力）**
   - エンドポイント: `POST /api/ai/photo-appraise`
   - 入力: `{ imageBase64, mimeType, condition }`
   - 画像から商品名・ブランド・カテゴリ・検索キーワードを識別
   - `generationConfig.responseMimeType: "application/json"` でJSON強制出力

2. **Step2 – Gemini + Google Search grounding**
   - Step1 の `searchKeyword` を使ってメルカリ・ヤフオク等の相場をリアル検索
   - ツール: `"tools": [{"googleSearch": {}}]`
   - レスポンス内の `<json>...</json>` タグを抽出してパース
   - 失敗時はカテゴリ別のデフォルト価格にフォールバック

3. 返却: `title, brand, category, price, minPrice, maxPrice, reason, searchSummary`

---

### 環境変数

- `GEMINI_API_KEY` — backend/.env に設定済み
- `GEMINI_MODEL` — backend/.env に設定済み（デフォルト: `gemini-2.0-flash`）

---

### フロントエンドの動作

- モバイル: `capture="environment"` により端末カメラが起動
- PC: ファイル選択ダイアログが開く
- 状態セレクト（未使用・未開封 / 未使用に近い / 良い / 普通 / 傷・汚れあり）
- 「フォームに反映する」ボタンで title・category・price・minPrice・description を自動入力
- カテゴリが既存 `CATEGORIES` リストに含まれない場合は手動選択を促すメッセージを表示
