package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/ltaoo/velo/frontendserver"
)

//go:embed frontend
var embeddedFrontend embed.FS

func main() {
	addr := flag.String("addr", "127.0.0.1:8090", "listen address")
	mode := flag.String("mode", "dev", "dev|prod")
	root := flag.String("root", "./frontend", "frontend root directory for dev mode")
	entry := flag.String("entry", "index.html", "entry page filename")
	flag.Parse()

	var srv http.Handler
	switch *mode {
	case "dev":
		srv = frontendserver.New(frontendserver.Options{
			Mode:      frontendserver.ModeDev,
			Root:      *root,
			EntryPage: *entry,
		})
	case "prod":
		srv = frontendserver.New(frontendserver.Options{
			Mode:      frontendserver.ModeProd,
			Root:      "frontend",
			Embedded:  embeddedFrontend,
			EntryPage: *entry,
		})
	default:
		log.Fatalf("unknown mode: %s (expected dev|prod)", *mode)
	}

	mux := http.NewServeMux()
	mux.Handle("/", srv)
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, `{"ok":true}`)
	})

	log.Printf("frontendserver demo: http://%s/ (mode=%s)", *addr, *mode)
	if *mode == "dev" {
		log.Printf("dev root: %s", *root)
	}
	log.Printf("try: http://%s/foo (SPA fallback), http://%s/api/ping", *addr, *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
