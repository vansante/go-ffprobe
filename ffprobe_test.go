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

	// Check some Tags
	const testLanguage = "und"
	if stream[0].Tags.Rotate != 0 {
		t.Errorf("Video stream rotate tag is not 0")
	}
	if stream[0].Tags.Language != testLanguage {
		t.Errorf("Video stream language tag is not %s", testLanguage)
	}

	if val, err := stream[0].TagList.GetString("language"); err != nil {
		t.Errorf("retrieving language tag errors: %v", err)
	} else if val != testLanguage {
		t.Errorf("Video stream language tag is not %s", testLanguage)
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

	stream = data.StreamType(StreamData)
	if len(stream) != 0 {
		t.Errorf("It does not have a data stream.")
	}

	stream = data.StreamType(StreamAttachment)
	if len(stream) != 0 {
		t.Errorf("It does not have an attachment stream.")
	}

	stream = data.StreamType(StreamAny)
	if len(stream) != 2 {
		t.Errorf("It should have two streams.")
	}

	// Check some Tags
	const testMajorBrand = "isom"
	if data.Format.Tags.MajorBrand != testMajorBrand {
		t.Errorf("MajorBrand format tag is not %s", testMajorBrand)
	}

	if val, err := data.Format.TagList.GetString("major_brand"); err != nil {
		t.Errorf("retrieving major_brand tag errors: %v", err)
	} else if val != testMajorBrand {
		t.Errorf("MajorBrand format tag is not %s", testMajorBrand)
	}

	// test Format.Duration
	duration := data.Format.Duration()
	if duration.Seconds() != 5.312 {
		t.Errorf("this video is 5.312s.")
	}
	// test Format.StartTime
	startTime := data.Format.StartTime()
	if startTime != time.Duration(0) {
		t.Errorf("this video starts at 0s.")
	}
}
