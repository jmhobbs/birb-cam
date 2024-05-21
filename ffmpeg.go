package main

import (
	"bufio"
	"context"
	"log"
	"fmt"
	"time"
	"os/exec"
	"path"
)

func hlsConvert(ctx context.Context, outputDir, rtspURL string) {
	cmd := exec.Command(
		"ffmpeg",
		// quieter output
		"-hide_banner", "-y",
		"-loglevel", "error",
		// reduce latency during input stream analysis
		"-fflags", "nobuffer",
		// rtsp over tcp
		"-rtsp_transport", "tcp",
		// heres our source
		"-i", rtspURL,
		// copy timestamps from rtsp stream
		"-copyts",
		// convert to h264 for firefox
		"-vcodec", "libx264",
		// output as fragment on each keyframe
		"-movflags", "frag_keyframe+empty_moov",
		// drop audio
		"-an",
		// fixed frame rate of 30
		"-r", "30",
		// set keyframe interval
		"-g", "30",
		"-keyint_min", "30",
		// HLS should delete own segments and append as they go
		"-hls_flags", "delete_segments+append_list",
		"-f", "hls",
		// try to make 3 second segments
		"-hls_time", "3",
		// keep 10 segments on disk
		"-hls_list_size", "5",
		"-hls_segment_type", "mpegts",
		"-hls_segment_filename", path.Join(outputDir, fmt.Sprintf("%d_%%d.ts", time.Now().Unix())),
		path.Join(outputDir, "index.m3u8"),
	)
	log.Println("Starting ffmpeg")

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Println(scanner.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	ffmpegDone := make(chan error)
	go func() {
		ffmpegDone <- cmd.Wait()
		close(ffmpegDone)
	}()

	select {
	case <-ctx.Done():
		log.Println("Cancelling ffmpeg")
		err := cmd.Process.Kill()
		if err != nil {
			log.Printf("unable to kill ffmpeg: %v", err)
		}
	case err = <-ffmpegDone:
		if err != nil {
			log.Printf("ffmpeg finished with error: %v", err)
		}
	}
}
