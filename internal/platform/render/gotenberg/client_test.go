package gotenberg

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConvertDocxToPDFPostsMultipartDOCX(t *testing.T) {
	docxContent := []byte("fake-docx-content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected method %s, got %s", http.MethodPost, r.Method)
		}
		if r.URL.Path != "/forms/libreoffice/convert" {
			t.Fatalf("expected path %q, got %q", "/forms/libreoffice/convert", r.URL.Path)
		}

		contentType := r.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
			t.Fatalf("expected multipart content type, got %q", contentType)
		}

		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("create multipart reader: %v", err)
		}
		part, err := reader.NextPart()
		if err != nil {
			t.Fatalf("read multipart part: %v", err)
		}
		if part.FormName() != "files" {
			t.Fatalf("expected form name %q, got %q", "files", part.FormName())
		}
		if part.FileName() != "document.docx" {
			t.Fatalf("expected file name %q, got %q", "document.docx", part.FileName())
		}

		payload, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("read multipart payload: %v", err)
		}
		if string(payload) != string(docxContent) {
			t.Fatalf("expected payload %q, got %q", string(docxContent), string(payload))
		}

		if _, err := reader.NextPart(); err != io.EOF {
			t.Fatalf("expected exactly one multipart part, got err=%v", err)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pdf-bytes"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.httpClient = server.Client()

	pdfContent, err := client.ConvertDocxToPDF(context.Background(), docxContent)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(pdfContent) != "pdf-bytes" {
		t.Fatalf("expected pdf response %q, got %q", "pdf-bytes", string(pdfContent))
	}
}

func TestConvertDocxToPDFReturnsStatusErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "conversion failed", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.httpClient = server.Client()

	_, err := client.ConvertDocxToPDF(context.Background(), []byte("fake-docx-content"))
	if err == nil {
		t.Fatalf("expected error when server returns non-200")
	}
	if !strings.Contains(err.Error(), "status 502: conversion failed") {
		t.Fatalf("expected status error with response body, got %v", err)
	}
}
