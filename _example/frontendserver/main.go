package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/ltaoo/velo/frontendserver"
)

/// go build -ldflags "-s -w -X main.Mode=prod" -o ./_example/frontendserver-demo ./_example/frontendserver
/// go build -ldflags "-s -w -X main.Mode=prod" -o frontendserver-demo ./main.go

var Mode = "dev"

//go:embed frontend
var embed_frontend embed.FS

func main() {
	addr := flag.String("addr", "127.0.0.1:8090", "listen address")
	flag.Parse()

	opt := frontendserver.Options{
		Mode:                Mode,
		Embedded:            embed_frontend,
		Root:                "frontend",
		EntryPage:           "index.html",
		StaticAssetPrefixes: []string{"/public", "/assets", "static/"},
		NoFallbackPrefixes:  []string{"/api"},
	}
	srv := frontendserver.New(opt)
	mux := http.NewServeMux()
	mux.Handle("/", srv)
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, `{"ok":true}`)
	})

	log.Printf("frontendserver demo: http://%s/ (mode=%s)", *addr, Mode)
	log.Printf("try: http://%s/foo (SPA fallback), http://%s/api/ping", *addr, *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
