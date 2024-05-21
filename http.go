package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options

//go:embed index.html
var index []byte

var connections map[string]*websocket.Conn = make(map[string]*websocket.Conn)

func countUpdater(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			count := []byte(strconv.Itoa(len(connections)))
			for _, conn := range connections {
				err := conn.WriteMessage(websocket.TextMessage, count)
				if err != nil {
					log.Printf("Error writing message to %s: %v", conn.RemoteAddr().String(), err)
				}
			}
		}
	}
}

func httpServer(ctx context.Context, outputDir string) {
	server := &http.Server{
		Addr: ":8080",
	}

	go countUpdater(ctx)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("error upgrading ws connection:", err)
			return
		}
		defer c.Close()

		connections[c.RemoteAddr().String()] = c
		defer delete(connections, c.RemoteAddr().String())

		// we never send client to server, so we just wait here
		// for a read, which should be the close notification
		c.ReadMessage()
	})

	http.HandleFunc("/index.mpd", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/dash+xml")
		w.Header().Set("Cache-Control", "max-age=5")
		http.ServeFile(w, r, path.Join(outputDir, "index.mpd"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Cache-Control", "max-age=3600")
			w.Write(index)
			return
		}

		if strings.HasSuffix(r.URL.Path, ".m4s") {
			w.Header().Set("Cache-Control", "max-age=30")
			http.ServeFile(w, r, path.Join(outputDir, r.URL.Path))
			return
		}

		http.NotFound(w, r)
	})

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
