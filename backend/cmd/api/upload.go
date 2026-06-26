package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

const maxImageBytes = 10 << 20

type uploadResult struct {
	PublicURL   string `json:"publicUrl"`
	ObjectPath  string `json:"objectPath"`
	ContentType string `json:"contentType"`
}

type imageStore interface {
	Save(context.Context, string, string, []byte) (uploadResult, error)
}

type localImageStore struct{ dir string }

func (s *localImageStore) Save(_ context.Context, purpose, contentType string, data []byte) (uploadResult, error) {
	name := fmt.Sprintf("%s-%d%s", purpose, time.Now().UnixNano(), imageExtension(contentType))
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return uploadResult{}, err
	}
	if err := os.WriteFile(filepath.Join(s.dir, name), data, 0o644); err != nil {
		return uploadResult{}, err
	}
	path := "/uploads/" + name
	return uploadResult{PublicURL: path, ObjectPath: path, ContentType: contentType}, nil
}

type gcsImageStore struct {
	client *storage.Client
	bucket string
}

func (s *gcsImageStore) Save(ctx context.Context, purpose, contentType string, data []byte) (uploadResult, error) {
	objectName := fmt.Sprintf("%s-images/%d%s", purpose, time.Now().UnixNano(), imageExtension(contentType))
	writer := s.client.Bucket(s.bucket).Object(objectName).NewWriter(ctx)
	writer.ContentType = contentType
	writer.CacheControl = "public, max-age=31536000, immutable"
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return uploadResult{}, err
	}
	if err := writer.Close(); err != nil {
		return uploadResult{}, err
	}
	objectPath := fmt.Sprintf("gcs://%s/%s", s.bucket, objectName)
	return uploadResult{
		PublicURL: "/uploads/" + objectName, ObjectPath: objectPath, ContentType: contentType,
	}, nil
}

func newImageStore(ctx context.Context) (imageStore, func(), error) {
	if bucket := strings.TrimSpace(os.Getenv("GCS_BUCKET")); bucket != "" {
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("initialize Cloud Storage: %w", err)
		}
		return &gcsImageStore{client: client, bucket: bucket}, func() { _ = client.Close() }, nil
	}
	dir := env("UPLOAD_DIR", "uploads")
	return &localImageStore{dir: dir}, func() {}, nil
}

func (a *app) uploadImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+(1<<20))
	if err := r.ParseMultipartForm(maxImageBytes + (1 << 20)); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "画像は1枚10MB以下にしてください")
		return
	}
	purpose := r.FormValue("purpose")
	if purpose != "item" && purpose != "avatar" {
		writeError(w, http.StatusBadRequest, "purpose must be item or avatar")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "画像ファイルを選択してください")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxImageBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "画像を読み込めませんでした")
		return
	}
	if len(data) == 0 || len(data) > maxImageBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "画像は1枚10MB以下にしてください")
		return
	}
	contentType := http.DetectContentType(data)
	if !allowedImageType(contentType) {
		writeError(w, http.StatusUnsupportedMediaType, "JPEG・PNG・WebP・GIF形式の画像を選択してください")
		return
	}
	if a.imageStore == nil {
		writeError(w, http.StatusServiceUnavailable, "画像保存サービスを利用できません")
		return
	}
	result, err := a.imageStore.Save(r.Context(), purpose, contentType, data)
	if err != nil {
		log.Printf("image upload failed: purpose=%s content_type=%s size=%d error=%v", purpose, contentType, len(data), err)
		writeError(w, http.StatusBadGateway, "画像の保存に失敗しました")
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (a *app) serveUploadedImage(w http.ResponseWriter, r *http.Request) {
	objectName := strings.TrimPrefix(r.URL.Path, "/uploads/")
	objectName = strings.TrimPrefix(filepath.Clean("/"+objectName), "/")
	if objectName == "." || objectName == "" || strings.HasPrefix(objectName, "../") {
		http.NotFound(w, r)
		return
	}

	if bucket := strings.TrimSpace(os.Getenv("GCS_BUCKET")); bucket != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()

		client, err := storage.NewClient(ctx)
		if err != nil {
			log.Printf("failed to initialize GCS reader: %v", err)
			http.Error(w, "画像を読み込めませんでした", http.StatusBadGateway)
			return
		}
		defer client.Close()

		obj := client.Bucket(bucket).Object(objectName)
		attrs, _ := obj.Attrs(ctx)
		reader, err := obj.NewReader(ctx)
		if err != nil {
			log.Printf("failed to read GCS object %s/%s: %v", bucket, objectName, err)
			http.NotFound(w, r)
			return
		}
		defer reader.Close()

		if attrs != nil && attrs.ContentType != "" {
			w.Header().Set("Content-Type", attrs.ContentType)
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		if _, err := io.Copy(w, reader); err != nil {
			log.Printf("failed to stream uploaded image %s: %v", objectName, err)
		}
		return
	}

	http.StripPrefix("/uploads/", http.FileServer(http.Dir(env("UPLOAD_DIR", "uploads")))).ServeHTTP(w, r)
}

func allowedImageType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func imageExtension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".jpg"
	}
}
