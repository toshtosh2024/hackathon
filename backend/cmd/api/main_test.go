package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCallOpenAIJSONParsesChatCompletionContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["response_format"] == nil {
			t.Fatal("expected response_format for JSON request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"prohibited\":true,\"riskLevel\":\"high\",\"reasons\":[\"刃物の可能性\"],\"blockedKeywords\":[\"ナイフ\"]}"}}]}`))
	}))
	defer server.Close()

	a := &app{
		openAIKey:     "test-key",
		openAIModel:   "gpt-test",
		openAIBaseURL: server.URL,
		httpClient:    server.Client(),
	}
	var review itemReview
	if err := a.callOpenAIJSON(context.Background(), "JSONで返して", &review); err != nil {
		t.Fatalf("callOpenAIJSON returned error: %v", err)
	}
	if !review.Prohibited || review.RiskLevel != "high" {
		t.Fatalf("unexpected review: %+v", review)
	}
	if len(review.BlockedKeywords) != 1 || review.BlockedKeywords[0] != "ナイフ" {
		t.Fatalf("unexpected keywords: %+v", review.BlockedKeywords)
	}
}

func TestExtractJSONObjectFromFencedText(t *testing.T) {
	got := extractJSONObject("```json\n{\"price\":1200}\n```")
	if got != `{"price":1200}` {
		t.Fatalf("unexpected JSON extraction: %q", got)
	}
}

func TestNormalizeRiskLevel(t *testing.T) {
	if got := normalizeRiskLevel("HIGH", false); got != "high" {
		t.Fatalf("expected high, got %q", got)
	}
	if got := normalizeRiskLevel("unknown", true); got != "high" {
		t.Fatalf("expected high fallback for prohibited item, got %q", got)
	}
	if got := normalizeRiskLevel("", false); got != "low" {
		t.Fatalf("expected low fallback, got %q", got)
	}
}

func TestClampPrice(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "minimum", in: 1, want: 300},
		{name: "normal", in: 4800, want: 4800},
		{name: "maximum", in: 99999999, want: 9999999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clampPrice(tt.in); got != tt.want {
				t.Fatalf("clampPrice(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestUploadPrefixSeparatesProfileImages(t *testing.T) {
	if got := uploadPrefix("avatar"); got != "avatars" {
		t.Fatalf("avatar prefix = %q, want avatars", got)
	}
	if got := uploadPrefix("item"); got != "items" {
		t.Fatalf("item prefix = %q, want items", got)
	}
	if got := uploadPrefix(""); got != "items" {
		t.Fatalf("default prefix = %q, want items", got)
	}
}

func TestTokenPreservesAvatarURL(t *testing.T) {
	a := &app{jwtSecret: "test-secret"}
	want := user{ID: 7, Name: "Toshi", Email: "toshi@example.com", Role: "user", AvatarURL: "https://example.com/avatar.png"}
	got, err := a.verifyToken(a.signToken(want))
	if err != nil {
		t.Fatalf("verifyToken returned error: %v", err)
	}
	if got.AvatarURL != want.AvatarURL {
		t.Fatalf("avatar URL = %q, want %q", got.AvatarURL, want.AvatarURL)
	}
}

func TestGuardDBReturnsDatabaseStatusWhenStarting(t *testing.T) {
	a := &app{}
	a.setDBStatus(context.DeadlineExceeded)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	a.guardDB(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run without a DB handle")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "database is starting" {
		t.Fatalf("unexpected error body: %+v", body)
	}
	database, ok := body["database"].(map[string]any)
	if !ok {
		t.Fatalf("expected database detail, got %+v", body["database"])
	}
	if database["ready"] != false {
		t.Fatalf("expected ready=false, got %+v", database)
	}
	if database["lastError"] == "" {
		t.Fatalf("expected lastError detail, got %+v", database)
	}
}

func TestResolveDSNUsesCloudSQLUnixSocket(t *testing.T) {
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("DB_USER", "nextmarket")
	t.Setenv("DB_PASS", "secret")
	t.Setenv("DB_NAME", "nextmarket")
	t.Setenv("INSTANCE_UNIX_SOCKET", "/cloudsql/project:asia-northeast1:next-market-mysql")
	t.Setenv("DB_HOST", "")

	got, err := resolveDSN()
	if err != nil {
		t.Fatalf("resolveDSN returned error: %v", err)
	}
	want := "nextmarket:secret@unix(/cloudsql/project:asia-northeast1:next-market-mysql)/nextmarket?parseTime=true&multiStatements=true"
	if got != want {
		t.Fatalf("resolveDSN = %q, want %q", got, want)
	}
}

func TestParseGCSRef(t *testing.T) {
	bucket, objectName, err := parseGCSRef("gcs://nextmarket/avatars/user-1.jpg")
	if err != nil {
		t.Fatalf("parseGCSRef returned error: %v", err)
	}
	if bucket != "nextmarket" || objectName != "avatars/user-1.jpg" {
		t.Fatalf("unexpected parse result: bucket=%q object=%q", bucket, objectName)
	}
}

func TestAssetViewURLKeepsPublicHTTPURL(t *testing.T) {
	got := assetViewURL("https://storage.googleapis.com/example/items/sample.jpg")
	if got != "https://storage.googleapis.com/example/items/sample.jpg" {
		t.Fatalf("assetViewURL changed public URL: %q", got)
	}
}

func TestLocalUploadURLSignsVerifiableToken(t *testing.T) {
	a := &app{jwtSecret: "test-secret"}
	req := httptest.NewRequest(http.MethodPost, "http://localhost:8080/api/upload", nil)
	uploadURL, publicURL, objectPath, err := a.localUploadURL(req, "items", "Sample Bag.JPG", "image/jpeg")
	if err != nil {
		t.Fatalf("localUploadURL returned error: %v", err)
	}
	if !strings.HasPrefix(uploadURL, "http://localhost:8080/api/local-upload?token=") {
		t.Fatalf("unexpected upload URL: %q", uploadURL)
	}
	if !strings.HasPrefix(publicURL, "http://localhost:8080/uploads/items/") {
		t.Fatalf("unexpected public URL: %q", publicURL)
	}
	if !strings.HasPrefix(objectPath, "local://items/") {
		t.Fatalf("unexpected object path: %q", objectPath)
	}

	parsed, err := url.Parse(uploadURL)
	if err != nil {
		t.Fatalf("parse upload URL: %v", err)
	}
	claim, err := a.verifyLocalUploadToken(parsed.Query().Get("token"))
	if err != nil {
		t.Fatalf("verifyLocalUploadToken returned error: %v", err)
	}
	if claim.ContentType != "image/jpeg" || !strings.HasPrefix(claim.Path, "items/") {
		t.Fatalf("unexpected claim: %+v", claim)
	}
}
