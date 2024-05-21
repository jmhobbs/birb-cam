package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os/exec"
	"path"
	"time"
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
		// scale to 1/2 source
		"-filter:v", "scale=960:-1",
		// fast, lossy encoding please
		"-preset", "superfast",
		"-crf", "40",
		// fast encoding, low latency streaming
		"-tune", "zerolatency",
		// output as fragment on each keyframe
		"-movflags", "frag_keyframe+empty_moov",
		// drop audio
		"-an",
		// ask to cap our bitrate
		"-maxrate", "500K",
		"-bufsize", "1M",
		// rate limit frames
		"-r", "15",
		// HLS should delete own segments and append as they go
		"-hls_flags", "delete_segments+append_list",
		"-f", "hls",
		// try to make 1 second segments
		"-hls_time", "1",
		// keep 5 segments on disk
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
