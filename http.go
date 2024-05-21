package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	_ "embed"
)

//go:embed index.html
var index []byte

func httpServer(ctx context.Context, outputDir string) {
	server := &http.Server{
		Addr: ":8080",
	}

	http.Handle("/index.m3u8", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-mpegURL")
		w.Header().Set("Cache-Control", "max-age=5")
		http.ServeFile(w, r, path.Join(outputDir, "index.m3u8"))
	}))

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Cache-Control", "max-age=3600")
			w.Write(index)
			return
		}

		if strings.HasSuffix(r.URL.Path, ".ts") {
			w.Header().Set("Content-Type", "video/mp2ts")
			w.Header().Set("Cache-Control", "max-age=3600")
			http.ServeFile(w, r, path.Join(outputDir, r.URL.Path))
			return
		}

		http.NotFound(w, r)
	}))

	go func() {
		log.Println("Starting HTTP server")
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
		log.Println("Stopped serving new connections.")
	}()

	<-ctx.Done()

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}

	log.Println("HTTP server stopped")
}
