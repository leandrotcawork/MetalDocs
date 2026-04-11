package gotenberg

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
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

func TestConvertHTMLToPDF_SendsMultipartToChromiumRoute(t *testing.T) {
	var capturedPath string
	var capturedBody []byte
	var capturedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		capturedBody = body
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("%PDF-1.4 fake"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.httpClient = server.Client()

	pdf, err := client.ConvertHTMLToPDF(
		context.Background(),
		[]byte("<html><body>Hi</body></html>"),
		[]byte("body { color: black; }"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("expected PDF magic bytes, got %q", string(pdf[:8]))
	}
	if capturedPath != "/forms/chromium/convert/html" {
		t.Fatalf("expected chromium route, got %q", capturedPath)
	}
	if !strings.HasPrefix(capturedContentType, "multipart/form-data") {
		t.Fatalf("expected multipart request, got %q", capturedContentType)
	}
	if !bytes.Contains(capturedBody, []byte("index.html")) {
		t.Fatalf("expected body to include index.html part")
	}
	if !bytes.Contains(capturedBody, []byte("style.css")) {
		t.Fatalf("expected body to include style.css part")
	}

	_, params, err := mime.ParseMediaType(capturedContentType)
	if err != nil {
		t.Fatalf("parse media type: %v", err)
	}
	mr := multipart.NewReader(bytes.NewReader(capturedBody), params["boundary"])
	seen := map[string]bool{}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		seen[part.FileName()] = true
	}
	if !seen["index.html"] || !seen["style.css"] {
		t.Fatalf("missing parts; saw %v", seen)
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
