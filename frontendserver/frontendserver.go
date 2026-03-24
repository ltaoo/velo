package frontendserver

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
)

type Mode int

const (
	ModeDev Mode = iota
	ModeProd
)

type Options struct {
	Mode      Mode
	Root      string
	Embedded  fs.FS
	EntryPage string
}

type Server struct {
	mode      Mode
	root      string
	entryPage string

	fileServer http.Handler
	indexBytes func() ([]byte, error)
	initErr    error
}

func New(opts Options) *Server {
	s := &Server{
		mode:      opts.Mode,
		root:      opts.Root,
		entryPage: opts.EntryPage,
	}
	if s.entryPage == "" {
		s.entryPage = "index.html"
	}

	switch s.mode {
	case ModeDev:
		if s.root == "" {
			s.initErr = os.ErrInvalid
			return s
		}
		s.fileServer = http.FileServer(http.Dir(s.root))
		s.indexBytes = func() ([]byte, error) {
			return os.ReadFile(filepath.Join(s.root, s.entryPage))
		}
		return s
	case ModeProd:
		if opts.Embedded == nil {
			s.initErr = os.ErrInvalid
			return s
		}
		root := s.root
		if root == "" {
			root = "frontend"
		}
		sub, err := fs.Sub(opts.Embedded, root)
		if err != nil {
			s.initErr = err
			return s
		}
		s.fileServer = http.FileServer(http.FS(sub))
		s.indexBytes = func() ([]byte, error) {
			return fs.ReadFile(sub, s.entryPage)
		}
		return s
	default:
		s.initErr = os.ErrInvalid
		return s
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.initErr != nil || s.fileServer == nil || s.indexBytes == nil {
		http.Error(w, "Frontend server not configured", http.StatusInternalServerError)
		return
	}

	isPublic := isPublicPath(r.URL.Path)
	if s.mode == ModeDev && !isPublic {
		setNoCacheHeaders(w.Header())
	}

	if r.URL.Path == "" || r.URL.Path == "/" {
		s.serveEntryPage(w)
		return
	}

	rec := httptest.NewRecorder()
	req := r
	if s.mode == ModeDev && !isPublic {
		req = r.Clone(r.Context())
		req.Header = r.Header.Clone()
		stripConditionalHeaders(req.Header)
	}
	s.fileServer.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound {
		if !isPublic {
			s.serveEntryPage(w)
			return
		}
	}

	for k, v := range rec.Result().Header {
		w.Header()[k] = v
	}
	if isPublic {
		setPublicCacheHeaders(w.Header())
	} else if s.mode == ModeDev {
		setNoCacheHeaders(w.Header())
	}
	w.WriteHeader(rec.Code)
	w.Write(rec.Body.Bytes())
}

func (s *Server) serveEntryPage(w http.ResponseWriter) {
	data, err := s.indexBytes()
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if s.mode == ModeDev {
		setNoCacheHeaders(w.Header())
	}
	w.Write(data)
}

func stripConditionalHeaders(h http.Header) {
	h.Del("If-Modified-Since")
	h.Del("If-None-Match")
	h.Del("If-Match")
	h.Del("If-Unmodified-Since")
	h.Del("If-Range")
}

func setNoCacheHeaders(h http.Header) {
	h.Set("Cache-Control", "no-store")
	h.Set("Pragma", "no-cache")
	h.Set("Expires", "0")
}

func setPublicCacheHeaders(h http.Header) {
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
}

func isPublicPath(p string) bool {
	if p == "" {
		return false
	}
	if strings.HasPrefix(p, "/public/") {
		return true
	}
	return p == "/public"
}
