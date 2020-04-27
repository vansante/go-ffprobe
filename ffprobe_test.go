package ffprobe

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	testPath = "assets/test.mp4"
)

func Test_ProbeURL(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFn()

	data, err := ProbeURL(ctx, testPath)
	if err != nil {
		t.Errorf("Error getting data: %v", err)
	}

	validateData(t, data)
}

func Test_ProbeURL_HTTP(t *testing.T) {
	const testPort = 20811

	ctx, cancelFn := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFn()

	// Serve all files in assets
	go func() {
		http.Handle("/", http.FileServer(http.Dir("./assets")))
		err := http.ListenAndServe(fmt.Sprintf(":%d", testPort), nil)
		t.Log(err)
	}()

	// Make sure HTTP is up
	time.Sleep(time.Second)

	data, err := ProbeURL(ctx, fmt.Sprintf("http://127.0.0.1:%d/test.mp4", testPort))
	if err != nil {
		t.Errorf("Error getting data: %v", err)
	}

	validateData(t, data)
}

func Test_ProbeReader(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFn()

	fileReader, err := os.Open(testPath)
	if err != nil {
		t.Errorf("Error opening test file: %v", err)
	}

	data, err := ProbeReader(ctx, fileReader)
	if err != nil {
		t.Errorf("Error getting data: %v", err)
	}

	validateData(t, data)
}

func validateData(t *testing.T, data *ProbeData) {
	// test ProbeData.GetStream
	stream := data.StreamType(StreamVideo)
	if len(stream) != 1 {
		t.Errorf("It just has one video stream.")
	}

	stream = data.StreamType(StreamAudio)
	if len(stream) != 1 {
		t.Errorf("It just has one audio stream.")
	}

	// this stream is []
	stream = data.StreamType(StreamSubtitle)
	if len(stream) != 0 {
		t.Errorf("It does not have a subtitle stream.")
	}

	stream = data.StreamType(StreamAny)
	if len(stream) != 2 {
		t.Errorf("It should have two streams.")
	}

	// test Format.Duration
	dration := data.Format.Duration()
	if dration.Seconds() != 5.312 {
		t.Errorf("this video is 5.312s.")
	}
	// test Format.StartTime
	startTime := data.Format.StartTime()
	if startTime != time.Duration(0) {
		t.Errorf("this video starts at 0s.")
	}
}
