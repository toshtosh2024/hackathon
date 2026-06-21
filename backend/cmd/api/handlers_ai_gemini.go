package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// geminiRequest は Gemini generateContent API のリクエストボディ構造体です。
type geminiRequest struct {
	Contents         []geminiContent  `json:"contents"`
	Tools            []geminiTool     `json:"tools,omitempty"`
	GenerationConfig *geminiGenConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiTool struct {
	GoogleSearch *struct{} `json:"google_search,omitempty"`
}

type geminiGenConfig struct {
	ResponseMimeType string `json:"responseMimeType,omitempty"`
}

// geminiResponse はGemini APIのレスポンスボディ
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (a *app) geminiModel() string {
	if m := os.Getenv("GEMINI_MODEL"); m != "" {
		return m
	}
	return "gemini-2.0-flash"
}

// geminiAPIKey は GEMINI_API_KEY または ANTIGRAVITY_API_KEY を getSecret 経由で取得します。
func (a *app) geminiAPIKey(ctx context.Context) string {
	if k := a.getSecret(ctx, "GEMINI_API_KEY"); k != "" {
		return k
	}
	return a.getSecret(ctx, "ANTIGRAVITY_API_KEY")
}

// callGeminiVision sends an image + text prompt to Gemini and returns a JSON string.
// Used for product identification from photos.
func (a *app) callGeminiVision(ctx context.Context, imageBase64, mimeType, prompt string) (string, error) {
	apiKey := a.geminiAPIKey(ctx)
	if apiKey == "" {
		return "", fmt.Errorf("missing GEMINI_API_KEY / ANTIGRAVITY_API_KEY")
	}

	reqBody := geminiRequest{
		Contents: []geminiContent{{
			Parts: []geminiPart{
				{InlineData: &geminiInlineData{MimeType: mimeType, Data: imageBase64}},
				{Text: prompt},
			},
		}},
	}

	return a.doGeminiRequest(ctx, apiKey, reqBody)
}

// callOpenAIVision sends an image + text prompt to OpenAI and returns a JSON string.
// Used as a fallback when Gemini is unavailable or has key configuration issues.
func (a *app) callOpenAIVision(ctx context.Context, imageBase64, mimeType, prompt string) (string, error) {
	apiKey := a.getSecret(ctx, "OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("missing OPENAI_API_KEY")
	}

	reqBody, err := json.Marshal(map[string]any{
		"model": "gpt-4o-mini",
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": prompt,
					},
					map[string]any{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:%s;base64,%s", mimeType, imageBase64),
						},
					},
				},
			},
		},
		"temperature": 0.2,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("OpenAI Vision API returned error %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", err
	}

	if len(res.Choices) == 0 {
		return "", errors.New("OpenAI Vision API returned empty response")
	}

	return res.Choices[0].Message.Content, nil
}

// callGeminiSearch sends a text prompt to Gemini with Google Search grounding enabled.
// Used for real-time market price research.
func (a *app) callGeminiSearch(ctx context.Context, prompt string) (string, error) {
	apiKey := a.geminiAPIKey(ctx)
	if apiKey == "" {
		return "", fmt.Errorf("missing GEMINI_API_KEY / ANTIGRAVITY_API_KEY")
	}

	reqBody := geminiRequest{
		Contents: []geminiContent{{
			Parts: []geminiPart{{Text: prompt}},
		}},
		Tools: []geminiTool{{GoogleSearch: &struct{}{}}},
	}

	return a.doGeminiRequest(ctx, apiKey, reqBody)
}

func (a *app) doGeminiRequest(ctx context.Context, apiKey string, reqBody geminiRequest) (string, error) {
	model := a.geminiModel()
	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		url.PathEscape(model), apiKey,
	)

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		errMsg := string(respBody)
		if len(errMsg) > 512 {
			errMsg = errMsg[:512] + "... (truncated)"
		}
		return "", fmt.Errorf("Gemini API error %d: %s", resp.StatusCode, errMsg)
	}

	var gemRes geminiResponse
	if err := json.Unmarshal(respBody, &gemRes); err != nil {
		return "", fmt.Errorf("failed to parse Gemini response: %w", err)
	}
	if len(gemRes.Candidates) == 0 || len(gemRes.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini returned empty response")
	}

	return gemRes.Candidates[0].Content.Parts[0].Text, nil
}

type photoCandidate struct {
	Title              string `json:"title"`
	Brand              string `json:"brand"`
	Category           string `json:"category"`
	EstimatedCondition string `json:"estimatedCondition,omitempty"`
	SearchKeyword      string `json:"searchKeyword,omitempty"`
	LikelihoodReason   string `json:"likelihoodReason,omitempty"`
	Price              int    `json:"price"`
	MinPrice           int    `json:"minPrice"`
	MaxPrice           int    `json:"maxPrice"`
	Reason             string `json:"reason"`
	SearchSummary      string `json:"searchSummary"`
}

func normalizePhotoCandidates(candidates []photoCandidate) []photoCandidate {
	if len(candidates) == 0 {
		candidates = []photoCandidate{
			{Title: "商品候補 1", Category: "その他", SearchKeyword: "フリマ 商品"},
			{Title: "商品候補 2", Category: "その他", SearchKeyword: "フリマ 商品"},
			{Title: "商品候補 3", Category: "その他", SearchKeyword: "フリマ 商品"},
		}
	}
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}
	for len(candidates) < 3 {
		base := candidates[0]
		base.Title = fmt.Sprintf("%s（候補%d）", base.Title, len(candidates)+1)
		candidates = append(candidates, base)
	}
	for i := range candidates {
		candidate := &candidates[i]
		if strings.TrimSpace(candidate.Title) == "" {
			candidate.Title = fmt.Sprintf("商品候補 %d", i+1)
		}
		if strings.TrimSpace(candidate.Category) == "" {
			candidate.Category = "その他"
		}
		if strings.TrimSpace(candidate.SearchKeyword) == "" {
			candidate.SearchKeyword = candidate.Title
		}
	}
	return candidates
}

// photoAppraise は写真からGemini Vision + Google Search grounding を用いて
// 商品候補を3件識別し、候補ごとの相場価格を提案するエンドポイントです。
func (a *app) photoAppraise(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ImageBase64 string `json:"imageBase64"`
		MimeType    string `json:"mimeType"`
		Condition   string `json:"condition"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.ImageBase64 == "" {
		writeError(w, http.StatusBadRequest, "imageBase64 is required")
		return
	}
	if req.MimeType == "" {
		req.MimeType = "image/jpeg"
	}
	if req.Condition == "" {
		req.Condition = "良い"
	}

	// Step 1: Gemini Vision で可能性の高い商品を3候補に絞る（検索なし・JSON出力）
	visionPrompt := fmt.Sprintf(`この商品画像を詳細に分析してください。

以下のJSONフォーマット"のみ"で出力してください（マークダウンや説明文は不要）：
{
  "candidates": [
    {
      "title": "商品名（ブランド・型番を含む具体的な名称）",
      "brand": "ブランド名（不明な場合は空文字）",
      "category": "以下のいずれか1つ: 家電・スマホ / 衣服・ファッション / 本・ゲーム・エンタメ / おもちゃ・ホビー / スポーツ・レジャー / ハンドメイド / その他",
      "estimatedCondition": "画像から見た状態の説明",
      "searchKeyword": "フリマサイト相場調査に使う最適な日本語検索キーワード",
      "likelihoodReason": "この候補と判断した画像上の根拠（40文字以内）"
    }
  ]
}

可能性が高い順に、異なる商品名・型番の候補を必ず3件返してください。

商品の状態（ユーザー申告）: %s`, req.Condition)

	var visionJSON string
	var geminiErr error

	apiKey := a.geminiAPIKey(r.Context())
	if apiKey != "" {
		log.Printf("Attempting Gemini Vision for photo appraisal...")
		visionJSON, geminiErr = a.callGeminiVision(r.Context(), req.ImageBase64, req.MimeType, visionPrompt)
	}

	if apiKey == "" || geminiErr != nil {
		if geminiErr != nil {
			log.Printf("Gemini Vision failed: %v. Falling back to OpenAI Vision...", geminiErr)
		} else {
			log.Printf("Gemini API key is missing. Falling back to OpenAI Vision...")
		}

		openAIKey := a.getSecret(r.Context(), "OPENAI_API_KEY")
		if openAIKey != "" {
			var openAIErr error
			visionJSON, openAIErr = a.callOpenAIVision(r.Context(), req.ImageBase64, req.MimeType, visionPrompt)
			if openAIErr != nil {
				var geminiErrMsg string
				if geminiErr != nil {
					geminiErrMsg = geminiErr.Error()
				} else {
					geminiErrMsg = "missing API key"
				}
				writeError(w, http.StatusBadGateway, fmt.Sprintf("AIによる商品識別に失敗しました。(Geminiエラー: %s / OpenAIフォールバックエラー: %v)", geminiErrMsg, openAIErr))
				return
			}
		} else {
			var geminiErrMsg string
			if geminiErr != nil {
				geminiErrMsg = geminiErr.Error()
			} else {
				geminiErrMsg = "missing API key"
			}
			writeError(w, http.StatusBadRequest, fmt.Sprintf("商品識別（Vision）に必要なAPIキーが設定されていません。Geminiキーが利用できず (%s)、OpenAIキーも設定されていません。ANTIGRAVITY_API_KEY、GEMINI_API_KEY、または OPENAI_API_KEY を設定してください。", geminiErrMsg))
			return
		}
	}

	// Step 1 のJSON をパース
	var visionResult struct {
		Candidates []photoCandidate `json:"candidates"`
	}
	cleanedJSON := extractJSON(visionJSON)
	if err := json.Unmarshal([]byte(cleanedJSON), &visionResult); err != nil {
		visionResult.Candidates = nil
	}
	visionResult.Candidates = normalizePhotoCandidates(visionResult.Candidates)

	// Step 2: Google Search grounding で3候補の日本のフリマ相場をまとめて調査
	candidateJSON, _ := json.Marshal(visionResult.Candidates)
	searchPrompt := fmt.Sprintf(`次の3つの商品候補について、日本のフリマアプリ（メルカリ、ヤフオク、ラクマ、PayPayフリマ等）の現在の取引相場をGoogle検索で調べてください。

商品候補: %s

商品状態（ユーザー申告）: %s

候補の順序を変えず、最後に以下のJSONブロックを出力してください：
<json>
{
  "candidates": [
    {
      "price": 推奨出品価格（数値・円）,
      "minPrice": 最低許容価格（数値・円）,
      "maxPrice": 市場最高価格の目安（数値・円）,
      "reason": "価格提案の根拠（市場調査結果を含む50文字以内）",
      "searchSummary": "相場調査サマリー（100文字以内）"
    }
  ]
}
</json>`, string(candidateJSON), req.Condition)

	var searchText string
	var err error
	if apiKey != "" {
		log.Printf("Attempting Gemini Search grounding for appraisal...")
		searchText, err = a.callGeminiSearch(r.Context(), searchPrompt)
	}

	if apiKey == "" || err != nil {
		if err != nil {
			log.Printf("Gemini Search failed: %v. Falling back to OpenAI for appraisal text...", err)
		} else {
			log.Printf("Gemini API key is missing. Falling back to OpenAI for appraisal text...")
		}

		openAIKey := a.getSecret(r.Context(), "OPENAI_API_KEY")
		if openAIKey != "" {
			searchText, err = a.callOpenAI(r.Context(), searchPrompt)
			if err != nil {
				searchText = "" // 失敗時はデフォルト相場にフォールバック
			}
		} else {
			searchText = ""
		}
	}

	// Step 2 の結果から JSON ブロックを抽出してパース
	var priceResult struct {
		Candidates []photoCandidate `json:"candidates"`
	}

	priceJSON := extractTaggedJSON(searchText)
	if priceJSON != "" {
		_ = json.Unmarshal([]byte(priceJSON), &priceResult)
	}

	for i := range visionResult.Candidates {
		candidate := &visionResult.Candidates[i]
		if i < len(priceResult.Candidates) {
			price := priceResult.Candidates[i]
			candidate.Price = price.Price
			candidate.MinPrice = price.MinPrice
			candidate.MaxPrice = price.MaxPrice
			candidate.Reason = price.Reason
			candidate.SearchSummary = price.SearchSummary
		}
		if candidate.Price <= 0 {
			candidate.Price = defaultPrice(candidate.Category, req.Condition)
			candidate.MinPrice = int(float64(candidate.Price) * 0.7)
			candidate.MaxPrice = int(float64(candidate.Price) * 1.5)
			candidate.Reason = "商品識別結果から概算価格を設定しました。"
			candidate.SearchSummary = searchText
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"condition":  req.Condition,
		"candidates": visionResult.Candidates,
	})
}

// extractJSON はGeminiのレスポンスから最初のJSONオブジェクトを抽出します。
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end <= start {
		return s
	}
	return s[start : end+1]
}

// extractTaggedJSON は <json>...</json> タグで囲まれたJSONを抽出します。
// タグがない場合は extractJSON にフォールバックします。
func extractTaggedJSON(s string) string {
	const openTag = "<json>"
	const closeTag = "</json>"
	start := strings.Index(s, openTag)
	end := strings.Index(s, closeTag)
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(s[start+len(openTag) : end])
	}
	return extractJSON(s)
}

// callGeminiImageGenerate は複数の入力画像とプロンプトから Gemini で合成画像を生成します。
// OpenAI の /v1/images/edits の代替として使用します。
func (a *app) callGeminiImageGenerate(ctx context.Context, prompt string, uploads []imageUpload) ([]byte, string, error) {
	apiKey := a.geminiAPIKey(ctx)
	if apiKey == "" {
		return nil, "", fmt.Errorf("missing GEMINI_API_KEY / ANTIGRAVITY_API_KEY")
	}

	// 入力画像を parts として組み立てる
	var parts []map[string]any
	for _, up := range uploads {
		parts = append(parts, map[string]any{
			"inlineData": map[string]string{
				"mimeType": up.ContentType,
				"data":     base64.StdEncoding.EncodeToString(up.Bytes),
			},
		})
	}
	parts = append(parts, map[string]any{"text": prompt})

	reqBody := map[string]any{
		"contents": []any{
			map[string]any{"parts": parts},
		},
		"generationConfig": map[string]any{
			"responseModalities": []string{"IMAGE"},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", err
	}

	// 画像生成は一般提供（GA）された gemini-3.1-flash-image を使用
	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-3.1-flash-image:generateContent?key=%s",
		apiKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("Gemini Image API error %d: %s", resp.StatusCode, string(respBody))
	}

	var gemRes struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text       string `json:"text"`
					InlineData *struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &gemRes); err != nil {
		return nil, "", fmt.Errorf("failed to parse Gemini image response: %w", err)
	}
	if len(gemRes.Candidates) == 0 {
		return nil, "", fmt.Errorf("Gemini returned no image candidates")
	}

	for _, part := range gemRes.Candidates[0].Content.Parts {
		if part.InlineData != nil {
			imgBytes, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
			return imgBytes, part.InlineData.MimeType, err
		}
	}

	return nil, "", fmt.Errorf("Gemini returned no image data in response")
}

// callGeminiVideoGenerate は Veo 3.1 (veo-3.1-generate-preview) モデルを叩き、
// 静止画から LRO (Long Running Operation) を使って動画（シネマグラフ）を生成し、完了をポーリングします。
func (a *app) callGeminiVideoGenerate(ctx context.Context, prompt string, base64Image string) ([]byte, error) {
	apiKey := a.geminiAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("missing GEMINI_API_KEY / ANTIGRAVITY_API_KEY")
	}

	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/veo-3.1-generate-preview:predictLongRunning?key=%s",
		apiKey,
	)

	payload := map[string]any{
		"instances": []any{
			map[string]any{
				"prompt": prompt,
				"image": map[string]any{
					"bytesBase64Encoded": base64Image,
				},
			},
		},
		"parameters": map[string]any{
			"aspectRatio":   "16:9",
			"videoDuration": 4,
			"sampleCount":   1,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// LRO 登録自体は一瞬なので 15 秒のタイムアウト
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Veo 3.1 API error %d: %s", resp.StatusCode, string(respBody))
	}

	var opRes struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(respBody, &opRes); err != nil {
		return nil, err
	}
	if opRes.Name == "" {
		return nil, fmt.Errorf("Veo 3.1 API returned no operation name")
	}

	operationName := opRes.Name
	pollURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s?key=%s", operationName, apiKey)

	log.Printf("Veo 3.1 video generation triggered. Operation name: %s. Starting polling...", operationName)

	// 最大 40 秒待機、5 秒おきにポーリング
	for i := 0; i < 8; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
		}

		pollReq, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
		if err != nil {
			return nil, err
		}

		pollResp, err := client.Do(pollReq)
		if err != nil {
			log.Printf("Veo 3.1 polling network error: %v. Retrying...", err)
			continue
		}

		pollBody, _ := io.ReadAll(pollResp.Body)
		pollResp.Body.Close()

		if pollResp.StatusCode >= 300 {
			log.Printf("Veo 3.1 polling error status %d: %s", pollResp.StatusCode, string(pollBody))
			continue
		}

		var opStatus struct {
			Done     bool `json:"done"`
			Response *struct {
				GenerateVideoResponse struct {
					GeneratedSamples []struct {
						Video struct {
							Uri                string `json:"uri"`
							BytesBase64Encoded string `json:"bytesBase64Encoded"`
						} `json:"video"`
					} `json:"generatedSamples"`
				} `json:"generateVideoResponse"`
				GeneratedVideos []struct {
					Video struct {
						Uri                string `json:"uri"`
						BytesBase64Encoded string `json:"bytesBase64Encoded"`
					} `json:"video"`
				} `json:"generatedVideos"`
				Predictions []struct {
					BytesBase64Encoded string `json:"bytesBase64Encoded"`
				} `json:"predictions"`
			} `json:"response"`
		}

		if err := json.Unmarshal(pollBody, &opStatus); err != nil {
			log.Printf("Veo 3.1 polling JSON unmarshal error: %v", err)
			continue
		}

		if opStatus.Done {
			if opStatus.Response == nil {
				return nil, fmt.Errorf("Veo 3.1 completed but returned no response payload")
			}

			// Gemini REST API の現在のレスポンス形式。
			if len(opStatus.Response.GenerateVideoResponse.GeneratedSamples) > 0 {
				video := opStatus.Response.GenerateVideoResponse.GeneratedSamples[0].Video
				if video.BytesBase64Encoded != "" {
					return base64.StdEncoding.DecodeString(video.BytesBase64Encoded)
				}
				if video.Uri != "" {
					return a.downloadGeminiVideo(ctx, client, video.Uri, apiKey)
				}
			}

			// bytesBase64Encoded が直接 predictions に含まれている場合
			if len(opStatus.Response.Predictions) > 0 && opStatus.Response.Predictions[0].BytesBase64Encoded != "" {
				return base64.StdEncoding.DecodeString(opStatus.Response.Predictions[0].BytesBase64Encoded)
			}

			// generatedVideos の中にある場合
			if len(opStatus.Response.GeneratedVideos) > 0 {
				video := opStatus.Response.GeneratedVideos[0].Video
				if video.BytesBase64Encoded != "" {
					return base64.StdEncoding.DecodeString(video.BytesBase64Encoded)
				}
				if video.Uri != "" {
					return a.downloadGeminiVideo(ctx, client, video.Uri, apiKey)
				}
			}

			return nil, fmt.Errorf("Veo 3.1 completed but found no base64 encoded video or uri")
		}
	}

	return nil, fmt.Errorf("Veo 3.1 video generation timed out (40s limit)")
}

func (a *app) downloadGeminiVideo(ctx context.Context, client *http.Client, uri, apiKey string) ([]byte, error) {
	downloadURL := uri
	if !strings.Contains(downloadURL, "?") {
		downloadURL = fmt.Sprintf("%s?key=%s", downloadURL, apiKey)
	} else {
		downloadURL = fmt.Sprintf("%s&key=%s", downloadURL, apiKey)
	}
	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	dlReq.Header.Set("x-goog-api-key", apiKey)
	dlResp, err := client.Do(dlReq)
	if err != nil {
		return nil, err
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode >= 300 {
		dlBody, _ := io.ReadAll(dlResp.Body)
		return nil, fmt.Errorf("failed to download video from URI %d: %s", dlResp.StatusCode, string(dlBody))
	}
	return io.ReadAll(dlResp.Body)
}

// defaultPrice はカテゴリーと状態に基づくフォールバック価格を返します。
func defaultPrice(category, condition string) int {
	base := map[string]int{
		"家電・スマホ":     15000,
		"衣服・ファッション":  3000,
		"本・ゲーム・エンタメ": 1500,
		"おもちゃ・ホビー":   2000,
		"スポーツ・レジャー":  4000,
		"ハンドメイド":     2500,
		"その他":        2000,
	}
	price, ok := base[category]
	if !ok {
		price = 2000
	}
	multiplier := map[string]float64{
		"未使用・未開封": 1.0,
		"未使用に近い":  0.85,
		"良い":      0.7,
		"普通":      0.55,
		"傷・汚れあり":  0.35,
	}
	if m, ok := multiplier[condition]; ok {
		price = int(float64(price) * m)
	}
	return price
}

// getSecret は指定された環境変数のキー(envKey)から値を取得し、
// もしその値がGCP Secret Managerのシークレットリソース名(projects/...)または
// シークレットID(60文字以下の単純な文字列で、かつGCPプロジェクト情報がある場合)
// である場合は、GCP Secret Manager APIから動的に本物のキーをフェッチします。
// 取得できない場合やローカル環境などでは、元の環境変数の生値をそのまま返します。
func (a *app) getSecret(ctx context.Context, envKey string) string {
	rawVal := strings.TrimSpace(os.Getenv(envKey))
	if rawVal == "" {
		return ""
	}

	var resourceName string
	if strings.HasPrefix(rawVal, "projects/") {
		resourceName = rawVal
		if !strings.Contains(resourceName, "/versions/") {
			resourceName = strings.TrimSuffix(resourceName, "/") + "/versions/latest"
		}
	} else {
		// APIキーは一般にランダムな長い文字列(記号など含む)ですが、シークレットIDは英数字・ハイフン・アンダースコアのみです。
		// 文字種チェックを行い、シークレットIDの形式に合致し、かつプロジェクトIDが取得できる場合のみSecret Managerを試行します。
		isSecretID := true
		for _, c := range rawVal {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
				isSecretID = false
				break
			}
		}

		if isSecretID && len(rawVal) < 60 {
			projectID := os.Getenv("FIRESTORE_PROJECT")
			if projectID == "" {
				projectID = os.Getenv("GCP_PROJECT")
			}
			if projectID == "" {
				projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
			}
			if projectID != "" {
				resourceName = fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, rawVal)
			}
		}
	}

	if resourceName == "" {
		return rawVal
	}

	log.Printf("Secret Manager: env %s points to secret %s. Attempting to fetch...", envKey, resourceName)
	token, err := a.getGCPToken()
	if err != nil {
		log.Printf("Secret Manager: Failed to get GCP identity token (falling back to raw env value): %v", err)
		return rawVal
	}

	reqURL := fmt.Sprintf("https://secretmanager.googleapis.com/v1/%s:access", resourceName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return rawVal
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// GCP API呼び出しなので5秒タイムアウト
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Secret Manager: HTTP request failed for %s: %v", resourceName, err)
		return rawVal
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Secret Manager: API returned status %d for %s: %s. Falling back.", resp.StatusCode, resourceName, string(body))
		return rawVal
	}

	var parsed struct {
		Payload struct {
			Data string `json:"data"`
		} `json:"payload"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		log.Printf("Secret Manager: Failed to parse API response: %v", err)
		return rawVal
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(parsed.Payload.Data))
	if err != nil {
		log.Printf("Secret Manager: Failed to decode base64 payload: %v", err)
		return rawVal
	}

	log.Printf("Secret Manager: Successfully fetched secret for env %s from GCP", envKey)
	return strings.TrimSpace(string(decoded))
}
