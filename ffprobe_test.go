package ffprobe

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	testPath      = "assets/test.mp4"
	testPathError = "assets/test.avi"
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

func Test_ProbeURL_Error(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFn()

	_, err := ProbeURL(ctx, testPathError, "-loglevel", "error")
	if err == nil {
		t.Errorf("No error reading bad asset")
	}

	if strings.Contains(err.Error(), "[]") {
		t.Errorf("No stderr included in error message")
	}
}

func Test_ProbeURL_HTTP(t *testing.T) {
	const testPort = 20811

	ctx, cancelFn := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFn()

	// Serve all files in assets
	go func() {
		http.Handle("/", http.FileServer(http.Dir("./assets")))
		host := ""
		if runtime.GOOS == "darwin" {
			host = "localhost" // macOS has security popups when not using localhost
		}

		err := http.ListenAndServe(fmt.Sprintf("%s:%d", host, testPort), nil) //nolint:gosec
		t.Log(err)
	}()

	// Make sure HTTP is up
	time.Sleep(time.Second)

	data, err := ProbeURL(ctx, fmt.Sprintf("http://127.0.0.1:%d/test.mp4", testPort))
	if err != nil {
		t.Errorf("Error getting data: %v", err)
	}

	validateData(t, data)

	_, err = ProbeURL(ctx, fmt.Sprintf("http://127.0.0.1:%d/test.avi", testPort), "-loglevel", "error")
	if err == nil {
		t.Errorf("No error reading bad asset")
	}

	if strings.Contains(err.Error(), "[]") {
		t.Errorf("No stderr included in error message")
	}
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

func Test_ProbeReader_Error(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFn()

	fileReader, err := os.Open(testPathError)
	if err != nil {
		t.Errorf("Error opening test file: %v", err)
	}

	_, err = ProbeReader(ctx, fileReader, "-loglevel", "error")
	if err == nil {
		t.Errorf("No error reading bad asset")
	}

	if strings.Contains(err.Error(), "[]") {
		t.Errorf("No stderr included in error message")
	}
}

func validateData(t *testing.T, data *ProbeData) {
	validateStreams(t, data)
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
	validateChapters(t, data)
}

func validateStreams(t *testing.T, data *ProbeData) {
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
	// We expect at least one data stream, since there are chapters
	if len(stream) == 0 {
		t.Errorf("It does not have a data stream.")
	}

	stream = data.StreamType(StreamAttachment)
	if len(stream) != 0 {
		t.Errorf("It does not have an attachment stream.")
	}

	stream = data.StreamType(StreamAny)
	if len(stream) != 3 {
		t.Errorf("It should have three streams.")
	}
}

func validateChapters(t *testing.T, data *ProbeData) {
	chapters := data.Chapters
	if chapters == nil {
		t.Error("Chapters List was nil")
		return
	}
	if len(chapters) != 3 {
		t.Errorf("Expected 3 chapters. Got %d", len(chapters))
		return
	}
	chapterToTest := chapters[1]
	if chapterToTest.Title() != "Middle" {
		t.Errorf("Bad Chapter Name. Got %s", chapterToTest.Title())
	}
	if chapterToTest.StartTimeSeconds != 2.0 {
		t.Errorf("Bad Chapter Start Time. Got %f", chapterToTest.StartTimeSeconds)
	}
	if chapterToTest.EndTimeSeconds != 4.0 {
		t.Errorf("Bad Chapter End Time. Got %f", chapterToTest.EndTimeSeconds)
	}
}

func Test_ProbeSideData(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFn()

	fileReader, err := os.Open("assets/test.mov")
	if err != nil {
		t.Errorf("Error opening test file: %v", err)
	}

	data, err := ProbeReader(ctx, fileReader)
	if err != nil {
		t.Errorf("Error getting data: %v", err)
	}

	videoStream := data.FirstVideoStream()
	if videoStream == nil {
		t.Error("Video Stream was nil")
		return
	}

	sideData, err := videoStream.SideDataList.GetDisplayMatrix()
	if err != nil {
		t.Errorf("Error getting display matrix: %v", err)
	}

	if sideData.Rotation != -180 {
		t.Errorf("Expected rotation to be -180, got %d", sideData.Rotation)
	}
}

func Test_runProbe_InvalidJSON(t *testing.T) {
	// Create a command that outputs invalid JSON
	ctx, cancelFn := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFn()

	// Use echo to output invalid JSON
	cmd := exec.CommandContext(ctx, "echo", "invalid json output")

	_, err := runProbe(cmd)
	if err == nil {
		t.Error("Expected error for invalid JSON output")
	}
	if !strings.Contains(err.Error(), "error parsing ffprobe output") {
		t.Errorf("Expected JSON parsing error, got: %v", err)
	}
}

func Test_runProbe_NoFormat(t *testing.T) {
	// Create a command that outputs valid JSON but without format data
	ctx, cancelFn := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFn()

	// Use echo to output JSON without format
	cmd := exec.CommandContext(ctx, "echo", `{"streams": []}`)

	_, err := runProbe(cmd)
	if err == nil {
		t.Error("Expected error for missing format data")
	}
	if !strings.Contains(err.Error(), "no format data found") {
		t.Errorf("Expected 'no format data found' error, got: %v", err)
	}
}
