package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"portaljuridico/internal/content"
	"portaljuridico/internal/ondemand"
)

func main() {
	addr := ":8080"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	repo, err := content.LoadRepository(".")
	if err != nil {
		log.Fatal(err)
	}
	generator := ondemand.New(repo, "var/on-demand-cache")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			w.Header().Set("X-Robots-Tag", "noindex,follow")
		}
		result, err := generator.RenderPath(r.URL.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if result.FromCache {
			w.Header().Set("X-Portal-Cache", "HIT")
		} else {
			w.Header().Set("X-Portal-Cache", "MISS")
		}
		fmt.Fprint(w, result.HTML)
	})

	log.Fatal(http.ListenAndServe(addr, mux))
}
