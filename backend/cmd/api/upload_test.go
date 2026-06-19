package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

type recordingImageStore struct {
	purpose     string
	contentType string
	data        []byte
}

func (s *recordingImageStore) Save(_ context.Context, purpose, contentType string, data []byte) (uploadResult, error) {
	s.purpose, s.contentType, s.data = purpose, contentType, append([]byte(nil), data...)
	return uploadResult{PublicURL: "/uploads/test.png", ObjectPath: "/uploads/test.png", ContentType: contentType}, nil
}

func TestUploadImageUsesDetectedContentType(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0, 0, 0, 0, 0}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("purpose", "item")
	part, err := writer.CreateFormFile("file", "misleading.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(png)
	_ = writer.Close()

	store := &recordingImageStore{}
	a := &app{imageStore: store}
	req := httptest.NewRequest(http.MethodPost, "/api/uploads", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	a.uploadImage(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if store.purpose != "item" || store.contentType != "image/png" {
		t.Fatalf("saved purpose=%q contentType=%q", store.purpose, store.contentType)
	}
	var response uploadResult
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.PublicURL != "/uploads/test.png" {
		t.Fatalf("publicUrl = %q", response.PublicURL)
	}
}

func TestUploadImageRejectsNonImage(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("purpose", "item")
	part, _ := writer.CreateFormFile("file", "notes.txt")
	_, _ = part.Write([]byte("not an image"))
	_ = writer.Close()

	a := &app{imageStore: &recordingImageStore{}}
	req := httptest.NewRequest(http.MethodPost, "/api/uploads", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	a.uploadImage(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestLocalImageStorePersistsFile(t *testing.T) {
	store := &localImageStore{dir: t.TempDir()}
	result, err := store.Save(context.Background(), "avatar", "image/png", []byte("png-data"))
	if err != nil {
		t.Fatal(err)
	}
	if result.ObjectPath == "" || result.PublicURL != result.ObjectPath || result.ContentType != "image/png" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
