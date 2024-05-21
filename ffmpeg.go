package main

import (
	"bufio"
	"context"
	"log"
	"os/exec"
	"path"
)

func hlsConvert(ctx context.Context, outputDir, rtspURL string) {
	cmd := exec.Command(
		"ffmpeg",
		// quieter output
		//"-hide_banner", "-y",
		//"-loglevel", "error",
		// heres our source
		"-i", rtspURL,
		// rtsp over tcp
		"-rtsp_transport", "tcp",
		"-tune", "zerolatency",
		"-c:v", "libx264",
		"-r", "20",
		// drop audio
		"-an",
		// output as fragment on each keyframe
		"-movflags", "frag_keyframe+empty_moov",
		// reduce latency during input stream analysis
		"-fflags", "nobuffer",
		"-f", "dash",
		"-window_size", "4",
		"-extra_window_size", "0",
		"-min_seg_duration", "2000000",
		"-remove_at_exit", "1",
		"-segment_wrap", "10",
		path.Join(outputDir, "index.mpd"),
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
