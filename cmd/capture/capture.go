// capture retrieves video streams from Nest cameras and feeds them to ffmpeg.
package main

/*

Copyright 2021 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/namsral/flag"

	"github.com/rspier/nestsdm"
	"google.golang.org/api/option"
	"google.golang.org/api/smartdevicemanagement/v1"
)

var (
	_             = flag.String("projectid", "", "project ID from https://console.nest.google.com/device-access/project-list (unused)")
	oAuthClientID = flag.String("oauth_clientid", "", "OAuth2 Client ID")
	oAuthSecret   = flag.String("oauth_secret", "", "OAuth2 Secret ID")

	deviceID    = flag.String("device", "", "Device ID to get video from")
	once        = flag.Bool("once", false, "run ffmpeg once, or forever")
	segmentTime = flag.Duration("segment_time", 24*60*60*time.Second, "size of ffmpeg segment")
	duration    = flag.Duration("duration", 0, "capture only this much video")
	fileSpec    = flag.String("file_spec", "/tmp/A-%Y%m%d-%H%M%S.mp4", "output filename, may contain strftime format directives")
	streamTo    = flag.String("stream_to", "", "rtsp URL to stream to")
)

func main() {
	flag.String(flag.DefaultConfigFlagname, "", "path to config file")
	flag.Parse()

	if *deviceID == "" {
		log.Fatalf("required flag --device not provided")
	}

	ctx := context.Background()

	outDir := filepath.Dir(*fileSpec)
	if !dirExists(outDir) {
		os.MkdirAll(outDir, 0700)
	}

	c := nestsdm.OAuthClient(*oAuthClientID, *oAuthSecret)
	sdm, err := smartdevicemanagement.NewService(context.Background(), option.WithHTTPClient(c))
	if err != nil {
		log.Fatalf("smartdevicemanagement.NewService(): %v", err)
	}

	args := captureArgs()
	if *streamTo != "" {
		args = streamArgs()
	}

	b := backoff.NewExponentialBackOff()
	repeat := true
	for repeat {
		// use backoff to try not to blow through all quota at once
		backoff.Retry(func() error {
			u, err := nestsdm.GenerateRTSPStream(ctx, sdm, *deviceID)
			if err != nil {
				log.Printf("getURL ERROR: %v", err)
				return fmt.Errorf("getURL ERROR: %v", err)
			}

			cctx, cxl := context.WithCancel(ctx)
			defer cxl()

			go nestsdm.Extender(cctx, sdm, u, *deviceID)

			err = ffmpeg(u.StreamURLs.RTSPURL, args)
			if err != nil {
				log.Printf("ffmpeg ERROR: %v", err)
				return fmt.Errorf("ffmpeg ERROR: %v", err)

			}
			repeat = !*once
			return nil
		}, b)
	}
}

func captureArgs() []string {
	args := []string{
		"-strftime", "1",
		"-f", "segment",
		"-segment_format", "mp4",
		"-segment_time", strconv.Itoa(int(segmentTime.Seconds())),
		"-segment_atclocktime", "1",
		"-reset_timestamps", "1",
		"-avoid_negative_ts", "1",
		"-map", "0",
	}
	if duration.Seconds() > 0 {
		args = append(args, "-t", strconv.Itoa(int(duration.Seconds())))
	}
	args = append(args,
		"-acodec", "copy", "-vcodec", "copy",
		*fileSpec)
	return args
}

func streamArgs() []string {
	return []string{
		"-acodec", "copy", "-vcodec", "copy",
		"-f", "flv",
		*streamTo,
	}
}

func ffmpeg(url string, args []string) error {

	// base args
	ffArgs := []string{
		"-rtsp_transport", "tcp",
		"-i", url,
		"-reconnect", "1",
	}
	ffArgs = append(ffArgs, args...)

	log.Printf("! ffmpeg %v", ffArgs)
	cmd := exec.Command("ffmpeg", ffArgs...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cmd.StderrPipe(): %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cmd.StdoutPipe(): %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cmd.Start(): %v", err)
	}

	go io.Copy(os.Stderr, stderr)
	go io.Copy(os.Stdout, stdout)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("cmd.Wait(): %v", err)
	}
	return nil
}

func dirExists(d string) bool {
	s, err := os.Stat(d)
	if err != nil {
		return false
	}
	if !s.IsDir() {
		return false
	}
	return true
}
