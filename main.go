package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sync"
	"syscall"

	_ "embed"
)

func main() {
	var (
		rtspURL      *string = flag.String("rtsp-url", "rtsp://192.168.1.124/1/h264major", "RTSP URL to stream from")
		hlsOutputDir *string = flag.String("hls-output-dir", "./", "Output directory for HLS files")
	)

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		hlsConvert(ctx, *hlsOutputDir, *rtspURL)
		cancel()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		httpServer(ctx, *hlsOutputDir)
	}()

	go func() {
		gracefulShutdown := make(chan os.Signal, 1)
		signal.Notify(gracefulShutdown, os.Interrupt, syscall.SIGTERM)
		<-gracefulShutdown
		cancel()
	}()

	wg.Wait()

	cleanup(*hlsOutputDir)

	log.Println("Goodbye")
}

func cleanup(outputDir string) {
	log.Println("Cleaning up HLS segments")

	err := os.Remove(path.Join(outputDir, "index.m3u8"))
	if err != nil {
		log.Printf("unable to remove \"index.m3u8\": %v", err)
	}

	tsFiles, err := filepath.Glob(path.Join(outputDir, "*.ts"))
	if err != nil {
		log.Println(err)
		return
	}

	for _, tsFile := range tsFiles {
		err = os.Remove(tsFile)
		if err != nil {
			log.Printf("unable to remove %q, %v:", tsFile, err)
		}
	}
}
