package frontendserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestDevMode_AlwaysServesLatestAndNoCache(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("index-v1"), 0644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.js"), []byte("v1"), 0644); err != nil {
		t.Fatalf("write app.js: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "public"), 0755); err != nil {
		t.Fatalf("mkdir public: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "public", "pub.js"), []byte("pub-v1"), 0644); err != nil {
		t.Fatalf("write public/pub.js: %v", err)
	}

	s := New(Options{
		Mode:      ModeDev,
		Root:      root,
		EntryPage: "index.html",
	})

	r1 := httptest.NewRequest(http.MethodGet, "http://example.local/app.js", nil)
	r1.Header.Set("If-Modified-Since", "Wed, 21 Oct 2015 07:28:00 GMT")
	w1 := httptest.NewRecorder()
	s.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w1.Code)
	}
	if got := w1.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", got)
	}
	if got := w1.Body.String(); got != "v1" {
		t.Fatalf("expected body v1, got %q", got)
	}

	pubReq := httptest.NewRequest(http.MethodGet, "http://example.local/public/pub.js", nil)
	pubReq.Header.Set("If-Modified-Since", "Wed, 21 Oct 2015 07:28:00 GMT")
	pubW := httptest.NewRecorder()
	s.ServeHTTP(pubW, pubReq)
	if pubW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", pubW.Code)
	}
	if got := pubW.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("expected public cache-control, got %q", got)
	}
	if got := pubW.Body.String(); got != "pub-v1" {
		t.Fatalf("expected body pub-v1, got %q", got)
	}

	if err := os.WriteFile(filepath.Join(root, "app.js"), []byte("v2"), 0644); err != nil {
		t.Fatalf("rewrite app.js: %v", err)
	}

	r2 := httptest.NewRequest(http.MethodGet, "http://example.local/app.js", nil)
	r2.Header.Set("If-Modified-Since", "Wed, 21 Oct 2015 07:28:00 GMT")
	w2 := httptest.NewRecorder()
	s.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w2.Code)
	}
	if got := w2.Body.String(); got != "v2" {
		t.Fatalf("expected body v2, got %q", got)
	}
}

func TestDevMode_SpaFallbackToEntryPage(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("index"), 0644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "public"), 0755); err != nil {
		t.Fatalf("mkdir public: %v", err)
	}

	s := New(Options{
		Mode:      ModeDev,
		Root:      root,
		EntryPage: "index.html",
	})

	r := httptest.NewRequest(http.MethodGet, "http://example.local/home/index", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("expected html content-type, got %q", got)
	}
	if got := w.Body.String(); got != "index" {
		t.Fatalf("expected index body, got %q", got)
	}

	r2 := httptest.NewRequest(http.MethodGet, "http://example.local/public/missing.js", nil)
	w2 := httptest.NewRecorder()
	s.ServeHTTP(w2, r2)

	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w2.Code)
	}
}

func TestProdMode_ServesEmbeddedFiles(t *testing.T) {
	embedded := fstest.MapFS{
		"frontend/index.html": {Data: []byte("index-prod")},
		"frontend/app.js":     {Data: []byte("prod")},
		"frontend/public/x.js": {Data: []byte("x")},
	}

	s := New(Options{
		Mode:      ModeProd,
		Root:      "frontend",
		Embedded:  embedded,
		EntryPage: "index.html",
	})

	r := httptest.NewRequest(http.MethodGet, "http://example.local/app.js", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	body, err := io.ReadAll(w.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "prod" {
		t.Fatalf("expected body prod, got %q", string(body))
	}
	if got := w.Header().Get("Cache-Control"); got == "no-store" {
		t.Fatalf("expected prod mode not to force no-store")
	}

	pubReq := httptest.NewRequest(http.MethodGet, "http://example.local/public/x.js", nil)
	pubW := httptest.NewRecorder()
	s.ServeHTTP(pubW, pubReq)
	if pubW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", pubW.Code)
	}
	if got := pubW.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("expected public cache-control, got %q", got)
	}

	pubMissReq := httptest.NewRequest(http.MethodGet, "http://example.local/public/missing.js", nil)
	pubMissW := httptest.NewRecorder()
	s.ServeHTTP(pubMissW, pubMissReq)
	if pubMissW.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", pubMissW.Code)
	}
}
